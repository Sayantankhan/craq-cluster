package craq

import (
	"context"
	"craq-cluster/gen/rpcpb"
	"io"
	"log"
	"os"
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeServer struct {
	rpcpb.UnimplementedNodeServer
	node *Node
}

func NewNodeServer(n *Node) *NodeServer {
	return &NodeServer{node: n}
}

func (s *NodeServer) QueryVersion(ctx context.Context, req *rpcpb.VersionQuery) (*rpcpb.VersionResponse, error) {
	internalReq := &rpcpb.VersionQuery{Folder: req.Folder, FileName: req.FileName}
	internalResp := &rpcpb.VersionResponse{}

	if err := s.node.HandleVersionQuery(internalReq, internalResp); err != nil {
		return nil, err
	}

	return &rpcpb.VersionResponse{
		Folder:   internalResp.Folder,
		Seq:      internalResp.Seq,
		FileName: internalResp.FileName,
		Path:     internalResp.Path,
	}, nil
}

func (s *NodeServer) StreamWrite(stream rpcpb.Node_StreamWriteServer) error {
	log.Println("[StreamWrite] ➡️ Starting to receive stream...")

	var (
		firstReq *rpcpb.StreamWriteReq
		tempFile *os.File
	)

	// Step 1: Read stream and write to /tmp/{chunkId}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break // done receiving chunks
		}
		if err != nil {
			log.Printf("[StreamWrite] ❌ Receive error: %v\n", err)
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		if firstReq == nil {
			firstReq = req

			// Construct full file path: /tmp/{folder}/{file_name}
			tempPath := filepath.Join("/tmp", req.Folder, req.FileName)
			firstReq.Path = tempPath

			// Ensure folder exists
			if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
				log.Printf("[StreamWrite] ❌ Failed to create folder: %v", err)
				return status.Errorf(codes.Internal, "mkdir failed: %v", err)
			}

			// Create the file
			tempFile, err = os.Create(tempPath)
			if err != nil {
				log.Printf("[StreamWrite] ❌ Failed to create file: %v\n", err)
				return status.Errorf(codes.Internal, "file create failed: %v", err)
			}
		}

		_, err = tempFile.Write(req.Data)
		if err != nil {
			log.Printf("[StreamWrite] ❌ Failed to write chunk to file: %v\n", err)
			return status.Errorf(codes.Internal, "file write failed: %v", err)
		}
		log.Printf("[StreamWrite] 📦 Wrote chunk %d bytes", len(req.Data))
	}
	if tempFile != nil {
		tempFile.Close()
	}

	if firstReq == nil {
		log.Println("[StreamWrite] ❌ No chunks received")
		return status.Error(codes.InvalidArgument, "no data received")
	}

	// Step 2: Build internalReq and internalAck (just like Write)
	internalReq := &rpcpb.StreamWriteReq{
		Folder:   firstReq.Folder,
		Seq:      firstReq.Seq,
		FileName: firstReq.FileName,
		Path:     firstReq.Path, // path of the reconstructed file
	}

	internalAck := &rpcpb.WriteAck{}

	if err := s.node.HandleWrite(internalReq, internalAck); err != nil {
		log.Printf("[StreamWrite] ❌ HandleWrite failed: %v\n", err)
		return status.Errorf(codes.Internal, "HandleWrite failed: %v", err)
	}

	log.Printf("[StreamWrite] ✅ Sending final ack: Folder=%s File=%s Seq=%d", internalAck.Folder, internalAck.FileName, internalAck.Seq)
	return stream.SendAndClose(internalAck)
}

func (s *NodeServer) StreamRead(req *rpcpb.StreamReadReq, stream rpcpb.Node_StreamReadServer) error {
	log.Printf("[StreamRead] 📥 Received request for Folder=%s Filename=%s", req.Folder, req.FileName)

	// Fetch the latest metadata for the requested chunk
	meta, found := s.node.Storage.GetLatest(req.Folder, req.FileName)
	if !found {
		log.Printf("[StreamRead] ❌ Folder %s File %s not found", req.Folder, req.FileName)
		return status.Errorf(codes.NotFound, "Folder %s File % not found", req.Folder, req.FileName)
	}

	file, err := os.Open(meta.Path)
	if err != nil {
		log.Printf("[StreamRead] ❌ Failed to open file %s: %v", meta.Path, err)
		return status.Errorf(codes.Internal, "failed to open chunk file: %v", err)
	}
	defer file.Close()

	const chunkSize = 64 * 1024 // 64 KB chunks
	buf := make([]byte, chunkSize)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("[StreamRead] ❌ Read error: %v", err)
			return status.Errorf(codes.Internal, "read error: %v", err)
		}

		chunk := &rpcpb.ReadChunk{
			Data: buf[:n],
		}

		if sendErr := stream.Send(chunk); sendErr != nil {
			log.Printf("[StreamRead] ❌ Send error: %v", sendErr)
			return status.Errorf(codes.Internal, "send error: %v", sendErr)
		}
	}

	log.Printf("[StreamRead] ✅ Completed streaming Folder=%s Filename=%s", req.Folder, req.FileName)
	return nil
}

func (s *NodeServer) ListFiles(ctx context.Context, req *rpcpb.FolderQuery) (*rpcpb.FileList, error) {
	log.Printf("[ListFiles] 📁 Listing files for folder: %s", req.Folder)

	fileNames, err := s.node.Storage.ListFilesInFolder(req.Folder)
	if err != nil {
		log.Printf("[ListFiles] ❌ Failed to list files: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to list files: %v", err)
	}

	return &rpcpb.FileList{
		FileNames: fileNames,
	}, nil

}

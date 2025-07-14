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

func (s *NodeServer) Write(ctx context.Context, req *rpcpb.WriteReq) (*rpcpb.WriteAck, error) {
	internalReq := &rpcpb.WriteReq{
		ChunkId:  req.ChunkId,
		Seq:      req.Seq,
		FileName: req.FileName,
		Path:     req.Path,
	}

	internalAck := &rpcpb.WriteAck{}

	if err := s.node.HandleWrite(internalReq, internalAck); err != nil {
		return nil, err
	}

	return &rpcpb.WriteAck{
		ChunkId: internalAck.ChunkId,
		Seq:     internalAck.Seq,
	}, nil
}

func (s *NodeServer) Read(ctx context.Context, req *rpcpb.ReadReq) (*rpcpb.ReadResponse, error) {
	internalReq := &rpcpb.ReadReq{ChunkId: req.ChunkId}
	internalResp := &rpcpb.ReadResponse{}

	if err := s.node.HandleRead(internalReq, internalResp); err != nil {
		return nil, err
	}

	return &rpcpb.ReadResponse{
		ChunkId:  internalResp.ChunkId,
		Seq:      internalResp.Seq,
		FileName: internalResp.FileName,
		Path:     internalResp.Path,
	}, nil
}

func (s *NodeServer) QueryVersion(ctx context.Context, req *rpcpb.VersionQuery) (*rpcpb.VersionResponse, error) {
	internalReq := &rpcpb.VersionQuery{ChunkId: req.ChunkId}
	internalResp := &rpcpb.VersionResponse{}

	if err := s.node.HandleVersionQuery(internalReq, internalResp); err != nil {
		return nil, err
	}

	return &rpcpb.VersionResponse{
		ChunkId:  internalResp.ChunkId,
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

			tempPath := filepath.Join("/tmp", req.ChunkId)
			firstReq.Path = tempPath

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
	internalReq := &rpcpb.WriteReq{
		ChunkId:  firstReq.ChunkId,
		Seq:      firstReq.Seq,
		FileName: firstReq.FileName,
		Path:     firstReq.Path, // path of the reconstructed file
	}
	internalAck := &rpcpb.WriteAck{}

	if err := s.node.HandleWrite(internalReq, internalAck); err != nil {
		log.Printf("[StreamWrite] ❌ HandleWrite failed: %v\n", err)
		return status.Errorf(codes.Internal, "HandleWrite failed: %v", err)
	}

	log.Printf("[StreamWrite] ✅ Sending final ack: ChunkId=%s Seq=%d", internalAck.ChunkId, internalAck.Seq)
	return stream.SendAndClose(internalAck)
}

func (s *NodeServer) StreamRead(req *rpcpb.StreamReadReq, stream rpcpb.Node_StreamReadServer) error {
	log.Printf("[StreamRead] 📥 Received request for chunk_id=%s", req.ChunkId)

	// Fetch the latest metadata for the requested chunk
	meta, found := s.node.Storage.GetLatest(req.ChunkId)
	if !found {
		log.Printf("[StreamRead] ❌ Chunk %s not found", req.ChunkId)
		return status.Errorf(codes.NotFound, "chunk %s not found", req.ChunkId)
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

	log.Printf("[StreamRead] ✅ Completed streaming chunk %s", req.ChunkId)
	return nil
}

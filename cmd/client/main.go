package main

import (
	"context"
	"craq-cluster/cmd/manager/gen/managerpb"
	"craq-cluster/gen/rpcpb"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <file_path> <fodler>", os.Args[0])
	}
	filePath := os.Args[1]
	folder := os.Args[2]
	fileName := filepath.Base(filePath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Connect to Manager
	mgrConn, err := grpc.Dial("localhost:9005", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Failed to connect to Manager: %v", err)
	}
	defer mgrConn.Close()

	mgrClient := managerpb.NewManagerClient(mgrConn)

	// Step 2: Ask Manager for head node
	writeHead, err := mgrClient.GetWriteHead(ctx, &managerpb.Empty{})
	if err != nil {
		log.Fatalf("❌ Manager.GetWriteHead failed: %v", err)
	}
	log.Printf("📤 Head node for write: %s (%s)", writeHead.NodeId, writeHead.Address)

	// Step 3: Connect to head node and stream file
	writeConn, err := grpc.Dial(writeHead.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Failed to dial write node: %v", err)
	}
	defer writeConn.Close()

	writeClient := rpcpb.NewNodeClient(writeConn)
	writeStream, err := writeClient.StreamWrite(ctx)
	if err != nil {
		log.Fatalf("❌ StreamWrite failed: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("❌ Failed to open file: %v", err)
	}
	defer file.Close()

	const chunkSize = 64 * 1024
	buf := make([]byte, chunkSize)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("❌ File read failed: %v", err)
		}

		err = writeStream.Send(&rpcpb.StreamWriteReq{
			Folder:   folder,
			Seq:      0,
			FileName: fileName,
			Path:     "", // server stores to /tmp/{chunkID}
			Data:     buf[:n],
		})
		if err != nil {
			log.Fatalf("❌ Send chunk failed: %v", err)
		}
	}
	ack, err := writeStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("❌ StreamWrite close failed: %v", err)
	}
	log.Printf("✅ Write complete: Folder=%s File=%s Seq=%d", ack.Folder, ack.FileName, ack.Seq)

	// Step 4: Ask manager for read node
	readResp, err := mgrClient.GetReadNode(ctx, &managerpb.ReadNodeQuery{
		ChunkId: folder,
	})
	if err != nil {
		log.Fatalf("❌ Manager.GetReadNode failed: %v", err)
	}
	log.Printf("📥 Read node: %s (%s)", readResp.NodeId, readResp.Address)

	// Step 5: Connect to chosen read node
	readConn, err := grpc.Dial(readResp.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Failed to connect to read node: %v", err)
	}
	defer readConn.Close()

	readClient := rpcpb.NewNodeClient(readConn)
	readStream, err := readClient.StreamRead(ctx, &rpcpb.StreamReadReq{
		Folder:   folder,
		FileName: fileName,
	})
	if err != nil {
		log.Fatalf("❌ StreamRead failed: %v", err)
	}

	var reconstructed []byte
	chunkNum := 0
	totalBytes := 0
	for {
		chunk, err := readStream.Recv()
		if err == io.EOF {
			log.Println("📦 [StreamRead] ✅ All chunks received")
			break
		}
		if err != nil {
			log.Fatalf("❌ receive chunk failed: %v", err)
		}

		chunkNum++
		totalBytes += len(chunk.Data)
		log.Printf("📦 Chunk #%d received (%d bytes)", chunkNum, len(chunk.Data))
		reconstructed = append(reconstructed, chunk.Data...)
	}

	fmt.Println("✅ Final Output:")
	fmt.Println(string(reconstructed))

	// Step 6: List from the folder
	reqQuery := &rpcpb.FolderQuery{Folder: folder}
	resp, err := readClient.ListFiles(ctx, reqQuery)
	if err != nil {
		log.Fatalf("❌ ListFiles failed: %v", err)
	}

	fmt.Printf("📂 Files in folder %s:\n", folder)
	for _, fileName := range resp.FileNames {
		fmt.Println("📄", fileName)
	}
}

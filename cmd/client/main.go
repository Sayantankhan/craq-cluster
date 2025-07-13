package main

import (
	"context"
	"craq-cluster/gen/rpcpb"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <file_path> <chunk_id>", os.Args[0])
	}
	filePath := os.Args[1]
	chunkID := os.Args[2]

	// Step 1: Validate and extract file name
	fileName := filepath.Base(filePath)

	// Step 2: Store the file to disk location
	destPath := filepath.Join("/tmp", chunkID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		log.Fatalf("failed to write file to chunk path: %v", err)
	}

	// Step 3: Connect to head node (n1) and send metadata
	writeConn, err := grpc.Dial("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("write gRPC dial failed: %v", err)
	}
	defer writeConn.Close()

	writeClient := rpcpb.NewNodeClient(writeConn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writeResp, err := writeClient.Write(ctx, &rpcpb.WriteReq{
		ChunkId:  chunkID,
		Seq:      1,
		FileName: fileName,
		Path:     destPath,
	})
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}

	log.Printf("✅ Write successful: Chunk=%s, Seq=%d", writeResp.ChunkId, writeResp.Seq)

	// Step 4: Connect to tail node (n3) and request chunk
	readConn, err := grpc.Dial("localhost:8003", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("read gRPC dial failed: %v", err)
	}
	defer readConn.Close()

	readClient := rpcpb.NewNodeClient(readConn)

	readResp, err := readClient.Read(ctx, &rpcpb.ReadReq{
		ChunkId: chunkID,
	})
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}

	fmt.Println("✅ Retrieved from tail node:")
	fmt.Printf("Chunk: %s | Seq: %d\n", readResp.ChunkId, readResp.Seq)
	fmt.Println("---- File Content ----")

	// Read the file content from the path (since only path is sent back)
	data, err = os.ReadFile(readResp.Path)
	if err != nil {
		log.Fatalf("failed to read content from path %s: %v", readResp.Path, err)
	}
	fmt.Println(string(data))
}

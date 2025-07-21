/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"craq-cluster/cmd/manager/gen/managerpb"
	"craq-cluster/gen/rpcpb"
)

var foldr string
var filePath string

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Create A file",
	Run: func(cmd *cobra.Command, args []string) {
		if foldr == "" || filePath == "" {
			log.Fatalf("❌ --folder, and --filepath are required")
		}

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
				Folder:   foldr,
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
	},
}

func init() {
	putCmd.Flags().StringVar(&foldr, "folder", "", "Folder to upload to in CRAQ")
	putCmd.Flags().StringVar(&filePath, "file", "", "Local file path to upload")

	rootCmd.AddCommand(putCmd)
}

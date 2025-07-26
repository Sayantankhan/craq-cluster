/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"craq-cluster/cmd/manager/gen/managerpb"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"craq-cluster/gen/rpcpb"
)

var file, fldr string

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the file",
	Run: func(cmd *cobra.Command, args []string) {
		if fldr == "" || file == "" {
			log.Fatalf("❌ --folder, and --file are required")
		}
		mgrConn, err := grpc.Dial("localhost:9005", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("❌ Failed to connect to Manager: %v", err)
		}
		defer mgrConn.Close()

		mgrClient := managerpb.NewManagerClient(mgrConn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		readResp, err := mgrClient.GetReadNode(ctx, &managerpb.ReadNodeQuery{
			ClientId: fldr,
		})
		if err != nil {
			log.Fatalf("❌ Manager.GetReadNode failed: %v", err)
		}
		log.Printf("📥 Read node: %s (%s)", readResp.NodeId, readResp.Address)

		readConn, err := grpc.Dial(readResp.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("❌ Failed to connect to read node: %v", err)
		}
		defer readConn.Close()

		readClient := rpcpb.NewNodeClient(readConn)
		readStream, err := readClient.StreamRead(ctx, &rpcpb.StreamReadReq{
			Folder:   fldr,
			FileName: file,
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
	},
}

func init() {
	getCmd.Flags().StringVar(&fldr, "folder", "", "Folder to upload to in CRAQ")
	getCmd.Flags().StringVar(&file, "file", "", "Local file path to upload")
	rootCmd.AddCommand(getCmd)
}

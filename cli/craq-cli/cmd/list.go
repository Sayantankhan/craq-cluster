/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"craq-cluster/cmd/manager/gen/managerpb"
	"craq-cluster/gen/rpcpb"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var folder string

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all files in a folder",
	Run: func(cmd *cobra.Command, args []string) {
		if folder == "" {
			log.Fatalf("❌ Folder must be provided using --folder")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mgrConn, err := grpc.Dial("localhost:9005", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("❌ Failed to connect to manager: %v", err)
		}
		defer mgrConn.Close()

		mgrClient := managerpb.NewManagerClient(mgrConn)

		readResp, err := mgrClient.GetReadNode(ctx, &managerpb.ReadNodeQuery{ChunkId: folder})
		if err != nil {
			log.Fatalf("❌ GetReadNode failed: %v", err)
		}
		readConn, err := grpc.Dial(readResp.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("❌ Failed to connect to read node: %v", err)
		}
		defer readConn.Close()

		readClient := rpcpb.NewNodeClient(readConn)
		resp, err := readClient.ListFiles(ctx, &rpcpb.FolderQuery{Folder: folder})
		if err != nil {
			log.Fatalf("❌ ListFiles failed: %v", err)
		}

		fmt.Printf("📂 Files in folder %s:\n", folder)
		for _, name := range resp.FileNames {
			if strings.HasSuffix(name, "/") {
				fmt.Println("📁", strings.TrimSuffix(name, "/"))
			} else {
				fmt.Println("📄", name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&folder, "folder", "f", "", "Folder name to list files from")
}

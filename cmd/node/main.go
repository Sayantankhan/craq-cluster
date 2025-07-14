package main

import (
	"context"
	"craq-cluster/internal/config"
	"craq-cluster/pkg/craq"
	"craq-cluster/pkg/storage"
	"log"
	"net"
	"os"

	rpcpb "craq-cluster/gen/rpcpb"

	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("NODE_ID not set")
	}

	cfg, err := config.Load("config/config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	thisNode, nextNode := cfg.FindNodeAndSuccessor(nodeID)
	if thisNode == nil {
		log.Fatalf("store init failed: %v", err)
	}

	// Connect to CockroachDB
	store, err := storage.NewCraqStore(ctx, cfg.DB.Addr)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}

	// Setup next client if not tail
	var next rpcpb.NodeClient
	if nextNode != nil {
		log.Printf("[Init] Dialing next node: %s", nextNode.Addr)
		conn, err := grpc.Dial(nextNode.Addr, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("dial error: %v", err)
		}
		next = rpcpb.NewNodeClient(conn)
	}

	// Create local node
	localNode := craq.NewNode(
		thisNode.ID,
		thisNode.IsHead,
		thisNode.IsTail,
		store,
		nil,  // Prev is unused
		next, // Next node in chain
	)

	// Start gRPC server
	lis, err := net.Listen("tcp", thisNode.Addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	rpcpb.RegisterNodeServer(grpcServer, craq.NewNodeServer(localNode))

	log.Printf("🚀 Node %s is up | Addr: %s | Head: %v | Tail: %v | Next: %v",
		thisNode.ID, thisNode.Addr, thisNode.IsHead, thisNode.IsTail,
		func() string {
			if nextNode != nil {
				return nextNode.ID
			}
			return "nil"
		}())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve failed: %v", err)
	}
}

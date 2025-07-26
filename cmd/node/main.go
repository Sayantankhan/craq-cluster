package main

import (
	"context"
	"craq-cluster/internal/config"
	"craq-cluster/pkg/craq"
	"craq-cluster/pkg/storage"
	"log"
	"net"
	"os"
	"time"

	managerpb "craq-cluster/cmd/manager/gen/managerpb"
	rpcpb "craq-cluster/gen/rpcpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	nodeID   = os.Getenv("NODE_ID")
	nodeAddr = os.Getenv("NODE_ADDRESS")
)

const (
	heartbeatInterval = 5 * time.Second
)

func main() {
	ctx := context.Background()

	if nodeID == "" || nodeAddr == "" {
		log.Fatal("NODE_ID, NODE_ADDRESS must be set")
		panic(0)
	}

	cfg, err := config.Load("config/config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	managerAddress := cfg.Manager

	// Connect to Manager
	managerConn, err := grpc.Dial(managerAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial manager: %v", err)
	}
	defer managerConn.Close()

	managerClient := managerpb.NewManagerClient(managerConn)

	// Step 1: Register Node
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[INIT] Registering with manager: ID=%s Addr=%s", nodeID, nodeAddr)
	_, err = managerClient.RegisterNode(ctx, &managerpb.NodeInfo{
		NodeId:  nodeID,
		Address: nodeAddr,
	})
	if err != nil {
		log.Fatalf("Failed to register node: %v", err)
	}
	log.Printf("✅ Registered with manager as %s", nodeID)

	// Step 2: Wait for chain to be finalized and then get Successor
	var next rpcpb.NodeClient
	var nextNodeAddr string
	isTail := false

	maxRetries := 30
	retryDelay := 2 * time.Second

	for retries := 0; retries < maxRetries; retries++ {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := managerClient.GetSuccessor(ctx2, &managerpb.SuccessorQuery{NodeId: nodeID})
		cancel2()

		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.FailedPrecondition {
				log.Printf("[WARN] Chain not finalized yet. Retrying in %v... (attempt %d/%d)", retryDelay, retries+1, maxRetries)
				time.Sleep(retryDelay)
				continue
			} else {
				log.Fatalf("Failed to get successor: %v", err)
			}
		}

		if resp.Address == "" {
			log.Printf("[INFO] No successor. This node is the tail.")
			isTail = true
		} else {
			nextNodeAddr = resp.Address
			log.Printf("[INFO] Successor address: %s", nextNodeAddr)

			conn, err := grpc.Dial(nextNodeAddr, grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Failed to connect to successor node: %v", err)
			}
			next = rpcpb.NewNodeClient(conn)
		}
		break
	}

	if next == nil && !isTail {
		log.Fatalf("Unable to retrieve successor after %d attempts. Exiting.", maxRetries)
	}

	ctx2, _ := context.WithTimeout(context.Background(), 5*time.Second)
	writeHead, err := managerClient.GetWriteHead(ctx2, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("❌ Manager.GetWriteHead failed: %v", err)
	}
	log.Printf("📤 Head node for write: %s (%s)", writeHead.NodeId, writeHead.Address)

	// Assume head is node1 for demo
	isHead := nodeID == writeHead.NodeId

	store, err := storage.NewCraqStore(context.Background(), cfg.DB.Addr)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}

	localNode := craq.NewNode(
		nodeID,
		isHead,
		isTail,
		store,
		nil,  // Prev not used
		next, // Next client
	)

	// Start gRPC Server
	lis, err := net.Listen("tcp", nodeAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", nodeAddr, err)
	}

	grpcServer := grpc.NewServer()
	rpcpb.RegisterNodeServer(grpcServer, craq.NewNodeServer(localNode))

	log.Printf("🚀 Node started | ID: %s | Addr: %s | Head: %v | Tail: %v | Next: %v",
		nodeID, nodeAddr, isHead, isTail,
		func() string {
			if nextNodeAddr != "" {
				return nextNodeAddr
			}
			return "nil"
		}())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve failed: %v", err)
	}
}

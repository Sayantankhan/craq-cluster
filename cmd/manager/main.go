package main

import (
	"context"
	"net"
	"sync"
	"time"

	"craq-cluster/cmd/manager/gen/managerpb"
	"craq-cluster/internal/config"

	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Manager struct {
	managerpb.UnimplementedManagerServer
	sync.RWMutex
	nodes        []config.NodeInfo
	nodeStatus   map[string]bool // nodeID -> isAlive
	tailIndex    int
	readCursor   int
	heartbeatInt time.Duration
}

func NewManager(initialNodes []config.NodeInfo) *Manager {
	m := &Manager{
		nodes:        initialNodes,
		nodeStatus:   make(map[string]bool),
		tailIndex:    findTailIndex(initialNodes),
		heartbeatInt: 3 * time.Second,
	}

	// start background heartbeat
	// go m.monitorHeartbeats()
	return m
}

func findTailIndex(nodes []config.NodeInfo) int {
	for i, node := range nodes {
		if node.IsTail {
			return i
		}
	}
	return -1
}

func (m *Manager) GetWriteHead(ctx context.Context, _ *managerpb.Empty) (*managerpb.NodeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	for _, n := range m.nodes {
		if n.IsHead {
			return &managerpb.NodeInfo{NodeId: n.ID, Address: n.Addr}, nil
		}
	}
	return nil, grpc.Errorf(codes.Unavailable, "No available head node")
}

func (m *Manager) GetReadNode(ctx context.Context, req *managerpb.ReadNodeQuery) (*managerpb.NodeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	// For now, if tail is healthy, always prefer it
	tail := m.nodes[m.tailIndex]
	if m.nodeStatus[tail.ID] {
		return &managerpb.NodeInfo{NodeId: tail.ID, Address: tail.Addr}, nil
	}

	// Else: round-robin on other healthy nodes
	n := m.nodes[m.readCursor%len(m.nodes)]
	m.readCursor++
	return &managerpb.NodeInfo{NodeId: n.ID, Address: n.Addr}, nil
}

func main() {
	// Load configuration
	cfg, err := config.Load("config/config.json")
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Initialize manager with nodes from config
	mgr := NewManager(cfg.Nodes)

	// Start gRPC server
	lis, err := net.Listen("tcp", ":9005")
	if err != nil {
		log.Fatalf("❌ Failed to listen on :9000: %v", err)
	}

	grpcServer := grpc.NewServer()
	managerpb.RegisterManagerServer(grpcServer, mgr)

	log.Println("🚀 Manager is running on :9005")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ gRPC serve failed: %v", err)
	}
}

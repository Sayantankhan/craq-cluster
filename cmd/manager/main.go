package main

import (
	"context"
	"craq-cluster/cmd/manager/gen/managerpb"
	"craq-cluster/internal/config"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Manager struct {
	managerpb.UnimplementedManagerServer
	sync.RWMutex

	nodes         map[string]config.NodeInfo
	nodeOrder     []string
	nodeStatus    map[string]bool
	expectedCount int
	chainBuilt    bool
}

func NewManager(expected int) *Manager {
	return &Manager{
		nodes:         make(map[string]config.NodeInfo),
		nodeOrder:     []string{},
		nodeStatus:    make(map[string]bool),
		expectedCount: expected,
	}
}

func (m *Manager) RegisterNode(ctx context.Context, req *managerpb.NodeInfo) (*emptypb.Empty, error) {
	m.Lock()
	defer m.Unlock()

	log.Printf("Node registration request: ID=%s, Addr=%s", req.GetNodeId(), req.GetAddress())

	if _, exists := m.nodes[req.NodeId]; !exists {
		m.nodeOrder = append(m.nodeOrder, req.NodeId)
	}

	m.nodes[req.NodeId] = config.NodeInfo{
		ID:   req.NodeId,
		Addr: req.Address,
	}

	m.nodeStatus[req.NodeId] = true

	log.Printf("📥 Registered node %s at %s", req.NodeId, req.Address)

	if len(m.nodes) == m.expectedCount && !m.chainBuilt {
		m.finalizeChain()
	}

	return &emptypb.Empty{}, nil
}

func (m *Manager) finalizeChain() {
	log.Println("🔗 Finalizing CRAQ chain...")

	for i, nodeID := range m.nodeOrder {
		node := m.nodes[nodeID]
		node.IsHead = (i == 0)
		node.IsTail = (i == len(m.nodeOrder)-1)
		m.nodes[nodeID] = node

		log.Printf(" - Node %s | Addr: %s | Head: %v | Tail: %v", node.ID, node.Addr, node.IsHead, node.IsTail)
	}

	m.chainBuilt = true
	log.Println("✅ Chain finalized!")
}

func (m *Manager) GetSuccessor(ctx context.Context, req *managerpb.SuccessorQuery) (*managerpb.NodeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	if !m.chainBuilt {
		return nil, status.Error(codes.FailedPrecondition, "Chain not finalized yet")
	}

	for i, id := range m.nodeOrder {
		if id == req.NodeId && i+1 < len(m.nodeOrder) {
			next := m.nodes[m.nodeOrder[i+1]]
			return &managerpb.NodeInfo{
				NodeId:  next.ID,
				Address: next.Addr,
				IsHead:  next.IsHead,
				IsTail:  next.IsTail,
			}, nil
		}
	}

	// No successor (probably tail)
	return &managerpb.NodeInfo{}, nil
}

func (m *Manager) Heartbeat(ctx context.Context, hb *managerpb.NodeHealth) (*emptypb.Empty, error) {
	m.Lock()
	defer m.Unlock()

	m.nodeStatus[hb.NodeId] = true
	return &emptypb.Empty{}, nil
}

func (m *Manager) GetWriteHead(ctx context.Context, _ *emptypb.Empty) (*managerpb.NodeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	if !m.chainBuilt || len(m.nodeOrder) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "Chain not finalized or no nodes registered")
	}

	headID := m.nodeOrder[0]
	headNode, ok := m.nodes[headID]
	if !ok {
		return nil, status.Errorf(codes.Internal, "Head node not found in registry")
	}

	return &managerpb.NodeInfo{
		NodeId:  headNode.ID,
		Address: headNode.Addr,
		IsHead:  true,
		IsTail:  false,
	}, nil
}

func (m *Manager) GetReadNode(ctx context.Context, _ *managerpb.ReadNodeQuery) (*managerpb.NodeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	if !m.chainBuilt || len(m.nodeOrder) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "Chain not finalized or no nodes registered")
	}

	idx := rand.Intn(len(m.nodeOrder))
	nodeID := m.nodeOrder[idx]
	node, ok := m.nodes[nodeID]
	if !ok {
		return nil, status.Errorf(codes.Internal, "Node %s not found in registry", nodeID)
	}

	return &managerpb.NodeInfo{
		NodeId:  node.ID,
		Address: node.Addr,
		IsHead:  idx == 0,
		IsTail:  idx == len(m.nodeOrder)-1,
	}, nil
}

func main() {
	// Optional: Enable file:line logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	expectedCountStr := os.Getenv("EXPECTED_NODE_COUNT")
	if expectedCountStr == "" {
		log.Fatal("EXPECTED_NODE_COUNT env not set")
	}
	expectedCount, err := strconv.Atoi(expectedCountStr)
	if err != nil || expectedCount <= 0 {
		log.Fatalf("Invalid EXPECTED_NODE_COUNT: %v", expectedCountStr)
	}

	manager := NewManager(expectedCount)

	// Start gRPC server
	listenAddr := ":9005"
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("❌ Failed to listen on %s: %v", listenAddr, err)
	}

	grpcServer := grpc.NewServer()
	managerpb.RegisterManagerServer(grpcServer, manager)

	log.Printf("🚀 Manager is running on %s | Expecting %d nodes", listenAddr, expectedCount)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ gRPC serve failed: %v", err)
	}
}

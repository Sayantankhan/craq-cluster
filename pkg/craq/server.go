package craq

import (
	"context"
	"craq-cluster/gen/rpcpb"
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

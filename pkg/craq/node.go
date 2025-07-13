package craq

import (
	"context"
	"craq-cluster/gen/rpcpb"
	"craq-cluster/pkg/storage"
	"fmt"
	"log"
	"sync"
)

type Node struct {
	ID      string
	IsHead  bool
	IsTail  bool
	Mutex   sync.Mutex
	Storage storage.StorageClient
	Prev    *NodeClient
	Next    *NodeClient
}

func NewNode(id string, isHead, isTail bool, store storage.StorageClient, prev, next *NodeClient) *Node {
	return &Node{
		ID:      id,
		IsHead:  isHead,
		IsTail:  isTail,
		Storage: store,
		Prev:    prev,
		Next:    next,
	}
}

func (n *Node) HandleWrite(req *rpcpb.WriteReq, ack *rpcpb.WriteAck) error {
	if n.IsHead {
		latest, found := n.Storage.GetLatest(req.ChunkId)
		if found {
			req.Seq = latest.Seq + 1
		} else {
			req.Seq = 1
		}
	}
	n.Mutex.Lock()

	// Store as dirty version locally
	if err := n.Storage.Put(req.ChunkId, req.Seq, req.FileName, req.Path); err != nil {
		n.Mutex.Unlock()
		return fmt.Errorf("Storage Put failed: %w", err)
	}

	n.Mutex.Unlock()

	if n.IsTail {
		// Tail node: mark clean and generate ack
		if err := n.Storage.MarkClean(req.ChunkId, req.Seq); err != nil {
			return fmt.Errorf("MarkClean failed at tail: %w", err)
		}

		ack.ChunkId = req.ChunkId
		ack.Seq = req.Seq
		return nil
	}

	// Not tail: forward to successor
	nextAck, err := n.Next.Write(context.Background(), req)
	if err != nil {
		return fmt.Errorf("forward Write to successor failed: %w", err)
	}

	n.Mutex.Lock()
	if err := n.Storage.MarkClean(nextAck.ChunkId, nextAck.Seq); err != nil {
		n.Mutex.Unlock()
		return fmt.Errorf("MarkClean after successor ack failed: %w", err)
	}
	n.Mutex.Unlock()

	// Propagate ack upward
	ack.ChunkId = nextAck.ChunkId
	ack.Seq = nextAck.Seq
	return nil
}

func (n *Node) HandleRead(req *rpcpb.ReadReq, resp *rpcpb.ReadResponse) error {
	n.Mutex.Lock()
	chunk, found := n.Storage.GetLatest(req.ChunkId)
	log.Printf("📖 Node %s received read for chunk %s (state: %v)", n.ID, req.ChunkId, chunk.State)
	n.Mutex.Unlock()

	if !found {
		return fmt.Errorf("chunk %s not found", req.ChunkId)
	}

	if chunk.State == storage.Clean {
		resp.ChunkId = req.ChunkId
		resp.Seq = chunk.Seq
		resp.FileName = chunk.FileName
		resp.Path = chunk.Path
		return nil
	}

	query := &rpcpb.VersionQuery{ChunkId: req.ChunkId}
	versionResp, err := n.Next.QueryVersion(context.Background(), query)
	if err != nil {
		return fmt.Errorf("version query failed: %w", err)
	}

	resp.ChunkId = versionResp.ChunkId
	resp.Seq = versionResp.Seq
	resp.FileName = versionResp.FileName
	resp.Path = versionResp.Path
	return nil
}

func (n *Node) HandleVersionQuery(req *rpcpb.VersionQuery, resp *rpcpb.VersionResponse) error {
	if !n.IsTail {
		return fmt.Errorf("version query must be handled by tail")
	}

	n.Mutex.Lock()
	chunk, found := n.Storage.GetLatest(req.ChunkId)
	log.Printf("🔍 Tail %s responding to version query for chunk %s", n.ID, req.ChunkId)
	n.Mutex.Unlock()

	if !found {
		return fmt.Errorf("chunk %s not found at tail", req.ChunkId)
	}
	if chunk.State != storage.Clean {
		return fmt.Errorf("chunk %s at tail is not clean yet", req.ChunkId)
	}

	resp.ChunkId = req.ChunkId
	resp.Seq = chunk.Seq
	resp.FileName = chunk.FileName
	resp.Path = chunk.Path
	return nil
}

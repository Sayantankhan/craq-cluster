package craq

import (
	"context"
	"craq-cluster/gen/rpcpb"
	"craq-cluster/pkg/storage"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type Node struct {
	ID      string
	IsHead  bool
	IsTail  bool
	Mutex   sync.Mutex
	Storage storage.StorageClient
	Prev    rpcpb.NodeClient
	Next    rpcpb.NodeClient
}

func NewNode(id string, isHead, isTail bool, store storage.StorageClient, prev, next rpcpb.NodeClient) *Node {
	return &Node{
		ID:      id,
		IsHead:  isHead,
		IsTail:  isTail,
		Storage: store,
		Prev:    prev,
		Next:    next,
	}
}

func (n *Node) HandleWrite(req *rpcpb.StreamWriteReq, ack *rpcpb.WriteAck) error {
	if n.IsHead {
		latest, found := n.Storage.GetLatest(req.Folder, req.FileName)
		if found {
			req.Seq = latest.Seq + 1
		} else {
			req.Seq = 1
		}
	}
	n.Mutex.Lock()

	// Store as dirty version locally
	if err := n.Storage.Put(req.Seq, req.FileName, req.Folder, req.Path); err != nil {
		n.Mutex.Unlock()
		return fmt.Errorf("Storage Put failed: %w", err)
	}

	n.Mutex.Unlock()

	if n.IsTail {
		// Tail node: mark clean and generate ack
		if err := n.Storage.MarkClean(req.Folder, req.FileName, req.Seq); err != nil {
			return fmt.Errorf("MarkClean failed at tail: %w", err)
		}

		ack.FileName = req.FileName
		ack.Folder = req.Folder
		ack.Seq = req.Seq
		return nil
	}

	// Not tail: forward to successor
	nextAck, err := n.streamFileToNext(req)
	if err != nil {
		return fmt.Errorf("forward Write to successor failed: %w", err)
	}

	n.Mutex.Lock()
	if err := n.Storage.MarkClean(nextAck.Folder, nextAck.FileName, nextAck.Seq); err != nil {
		n.Mutex.Unlock()
		return fmt.Errorf("MarkClean after successor ack failed: %w", err)
	}
	n.Mutex.Unlock()

	// Propagate ack upward
	ack.FileName = nextAck.FileName
	ack.Folder = nextAck.Folder
	ack.Seq = nextAck.Seq
	return nil
}

func (n *Node) HandleVersionQuery(req *rpcpb.VersionQuery, resp *rpcpb.VersionResponse) error {
	if !n.IsTail {
		return fmt.Errorf("version query must be handled by tail")
	}

	n.Mutex.Lock()
	chunk, found := n.Storage.GetLatest(req.Folder, req.FileName)
	log.Printf("🔍 Tail %s responding to version query for Folder %s File %s", n.ID, req.Folder, req.FileName)
	n.Mutex.Unlock()

	if !found {
		return fmt.Errorf("Folder %s File %s not found at tail", req.Folder, req.FileName)
	}
	if chunk.State != storage.Clean {
		return fmt.Errorf("Folder %s File %s at tail is not clean yet", req.Folder, req.FileName)
	}

	resp.Folder = chunk.Folder
	resp.Seq = chunk.Seq
	resp.FileName = chunk.FileName
	resp.Path = chunk.Path
	return nil
}

func (n *Node) streamFileToNext(req *rpcpb.StreamWriteReq) (*rpcpb.WriteAck, error) {
	stream, err := n.Next.StreamWrite(context.Background())
	if err != nil {
		return nil, fmt.Errorf("start stream to next node failed: %w", err)
	}

	file, err := os.Open(req.Path)
	if err != nil {
		return nil, fmt.Errorf("open file failed: %w", err)
	}
	defer file.Close()

	const chunkSize = 64 * 1024
	buf := make([]byte, chunkSize)

	for {
		nBytes, readErr := file.Read(buf)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read file failed: %w", readErr)
		}

		err = stream.Send(&rpcpb.StreamWriteReq{
			Folder:   req.Folder,
			Seq:      req.Seq,
			FileName: req.FileName,
			Path:     req.Path,
			Data:     buf[:nBytes],
		})
		if err != nil {
			return nil, fmt.Errorf("send chunk failed: %w", err)
		}
	}

	ack, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("stream close failed: %w", err)
	}

	return ack, nil
}

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type DiskStore struct {
	baseDir string
	mu      sync.Mutex
	meta    map[string]Chunk // In-memory metadata
}

func NewDiskStore(baseDir string) (*DiskStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &DiskStore{
		baseDir: baseDir,
		meta:    make(map[string]Chunk),
	}, nil
}

func (s *DiskStore) Put(chunkId string, seq uint64, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileName := fmt.Sprintf("%s.chunk", chunkId)
	path := filepath.Join(s.baseDir, fileName)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	s.meta[chunkId] = Chunk{
		Seq:      seq,
		State:    Dirty,
		Path:     path,
		FileName: fileName,
	}
	return nil
}

func (s *DiskStore) MarkClean(chunkId string, seq uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chunk, ok := s.meta[chunkId]
	if !ok || chunk.Seq != seq {
		return fmt.Errorf("chunk not found or seq mismatch")
	}
	chunk.State = Clean
	s.meta[chunkId] = chunk
	return nil
}

func (s *DiskStore) GetLatest(chunkId string) (Chunk, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chunk, ok := s.meta[chunkId]
	return chunk, ok
}

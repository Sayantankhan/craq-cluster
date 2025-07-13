package storage

import (
	"fmt"
)

type HybridStore struct {
	Disk *DiskStore
	DB   *CraqStore
}

func NewHybridStore(disk *DiskStore, db *CraqStore) *HybridStore {
	return &HybridStore{Disk: disk, DB: db}
}

func (h *HybridStore) Put(chunkId string, seq uint64, data []byte) error {
	// Step 1: Write to disk
	err := h.Disk.Put(chunkId, seq, data)
	if err != nil {
		return fmt.Errorf("disk write failed: %w", err)
	}

	// Step 2: Lookup path info from Disk
	chunk, ok := h.Disk.GetLatest(chunkId)
	if !ok {
		return fmt.Errorf("disk metadata missing after write")
	}

	// Step 3: Save metadata to DB
	if err := h.DB.Put(chunkId, seq, chunk.FileName, chunk.Path); err != nil {
		return fmt.Errorf("db Put failed: %w", err)
	}

	return nil
}

func (h *HybridStore) MarkClean(chunkId string, seq uint64) error {
	err := h.Disk.MarkClean(chunkId, seq)
	if err != nil {
		return err
	}

	// Update DB state
	return h.DB.MarkClean(chunkId, seq)
}

func (h *HybridStore) GetLatest(chunkId string) (Chunk, bool) {
	return h.Disk.GetLatest(chunkId)
}

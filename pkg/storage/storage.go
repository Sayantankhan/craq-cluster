package storage

// VersionState represents whether a chunk is dirty or clean.
type VersionState int

const (
	Dirty VersionState = iota
	Clean
)

// Chunk stores metadata and data for a versioned chunk.
type Chunk struct {
	ChunkID  string       // UUID
	FileName string       // Original file name
	Seq      uint64       // Version/sequence number
	State    VersionState // "clean" or "dirty"
	Path     string       // Path to the chunk file on disk
}

type StorageClient interface {
	Put(chunkId string, seq uint64, path string, fileName string) error
	MarkClean(chunkId string, seq uint64) error
	GetLatest(chunkId string) (Chunk, bool)
}

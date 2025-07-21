package storage

// VersionState represents whether a chunk is dirty or clean.
type VersionState int

const (
	Dirty VersionState = iota
	Clean
)

// Chunk stores metadata and data for a versioned chunk.
type Chunk struct {
	Folder   string       // Folder
	FileName string       // Original file name
	Seq      uint64       // Version/sequence number
	State    VersionState // "clean" or "dirty"
	Path     string       // Path to the chunk file on disk
}

type StorageClient interface {
	Put(seq uint64, fileName, folder, path string) error
	MarkClean(folder, fileName string, seq uint64) error
	GetLatest(folder, fileName string) (Chunk, bool)
	ListFilesInFolder(folder string) ([]string, error)
}

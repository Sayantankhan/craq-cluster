package util

import (
	"craq-cluster/gen/rpcpb"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const ChunkSize = 4 * 1024 * 1024 // 4MB

func ChunkFile(filePath string) ([]rpcpb.WriteReq, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(filePath)
	chunkID := uuid.New().String()
	var chunks []rpcpb.WriteReq

	for i := 0; i*ChunkSize < len(data); i++ {
		start := i * ChunkSize
		end := (i + 1) * ChunkSize
		if end > len(data) {
			end = len(data)
		}

		fmt.Printf(fileName + " " + chunkID + " " + string(start))
	}
	return chunks, nil
}

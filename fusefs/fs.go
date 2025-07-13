package fusefs

import (
	"context"
	"craq-cluster/gen/rpcpb"
	"fmt"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/google/uuid"
)

type FS struct {
	MountDir string
	Node     rpcpb.NodeClient
}

func (f *FS) Root() (fs.Node, error) {
	return &Dir{fs: f}, nil
}

type Dir struct {
	fs *FS
}

type File struct {
	fs       *FS
	chunkID  string
	fileName string
	path     string
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0755
	return nil
}

func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	chunkID := uuid.New().String()
	localPath := filepath.Join(d.fs.MountDir, "chunks", chunkID)

	// Ensure chunks dir exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create chunks dir: %w", err)
	}

	// Create empty file on disk
	file, err := os.Create(localPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create local file: %w", err)
	}
	file.Close()

	// Notify backend
	_, err = d.fs.Node.Write(ctx, &rpcpb.WriteReq{
		ChunkId:  chunkID,
		FileName: req.Name,
		Path:     localPath,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("rpc write failed: %w", err)
	}

	node := &File{
		fs:       d.fs,
		chunkID:  chunkID,
		fileName: req.Name,
		path:     localPath,
	}
	return node, node, nil
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	info, err := os.Stat(f.path)
	if err != nil {
		return err
	}
	a.Mode = 0644
	a.Size = uint64(info.Size())
	a.Mtime = info.ModTime()
	a.Ctime = info.ModTime()
	return nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	file, err := os.OpenFile(f.path, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file for write failed: %w", err)
	}
	defer file.Close()

	// Write at offset
	n, err := file.WriteAt(req.Data, req.Offset)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	resp.Size = n

	// Mark chunk as dirty in backend
	_, err = f.fs.Node.Write(ctx, &rpcpb.WriteReq{
		ChunkId:  f.chunkID,
		FileName: f.fileName,
		Path:     f.path,
	})
	if err != nil {
		return fmt.Errorf("rpc write failed: %w", err)
	}
	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	return data, nil
}

package vfs

import (
	"bytes"
	"io/fs"
	"time"
)

// ByteFile implements fs.File over a byte slice.
type ByteFile struct {
	*bytes.Reader
}

// NewByteFile creates a new ByteFile.
func NewByteFile(data []byte) *ByteFile {
	return &ByteFile{Reader: bytes.NewReader(data)}
}

// Close implements fs.File Close method (no-op for a byte slice).
func (f *ByteFile) Close() error {
	return nil
}

// Stat returns the FileInfo structure describing file.
// As it's a simple file, we return a dummy static stat result.
func (f *ByteFile) Stat() (fs.FileInfo, error) {
	return &byteFileInfo{size: int64(f.Len())}, nil
}

// byteFileInfo implements fs.FileInfo interface for a ByteFile.
type byteFileInfo struct {
	size int64
}

func (fi *byteFileInfo) Name() string {
	return "bytefile"
}

func (fi *byteFileInfo) Size() int64 {
	return fi.size
}

func (fi *byteFileInfo) Mode() fs.FileMode {
	return 0444 // read-only
}

func (fi *byteFileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *byteFileInfo) IsDir() bool {
	return false
}

func (fi *byteFileInfo) Sys() interface{} {
	return nil
}

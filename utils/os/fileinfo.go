package os

import (
	"io/fs"
	"time"
)

// FileInfo .
type FileInfo struct {
	FileName    string
	FileSize    int64
	FileMode    fs.FileMode
	FileModTime time.Time
	Dir         bool
	FileSys     interface{}
}

// Name .
func (f FileInfo) Name() string {
	return f.FileName
}

// Size .
func (f FileInfo) Size() int64 {
	return f.FileSize
}

// Mode .
func (f FileInfo) Mode() fs.FileMode {
	return f.FileMode
}

// ModTime .
func (f FileInfo) ModTime() time.Time {
	return f.FileModTime
}

// IsDir .
func (f FileInfo) IsDir() bool {
	return f.Dir
}

// Sys .
func (f FileInfo) Sys() interface{} {
	return f.FileSys
}

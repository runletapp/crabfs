package crabfs

import (
	"os"
	"time"
)

// FileInfo implements os.FileInfo interface
type FileInfo struct {
	os.FileInfo

	name  string
	size  int64
	mode  uint32
	mtime time.Time
}

// Name base name of the file
func (fileInfo *FileInfo) Name() string {
	return fileInfo.name
}

// Size length in bytes for regular files; system-dependent for others
func (fileInfo *FileInfo) Size() int64 {
	return fileInfo.size
}

// Mode file mode bits
func (fileInfo *FileInfo) Mode() os.FileMode {
	return os.FileMode(fileInfo.mode)
}

// ModTime modification time
func (fileInfo *FileInfo) ModTime() time.Time {
	return fileInfo.mtime
}

// IsDir abbreviation for Mode().IsDir()
func (fileInfo *FileInfo) IsDir() bool {
	return false
}

// Sys underlying data source (can return nil)
func (fileInfo *FileInfo) Sys() interface{} {
	return nil
}

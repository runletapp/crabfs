package interfaces

import (
	"context"
	"io"
	"time"
)

// Core interface
type Core interface {
	// Get opens a file stream
	Get(ctx context.Context, filename string) (io.ReadSeeker, int64, error)

	// Put writes a file to the storage
	Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error

	// Remove deletes a file from the storage
	Remove(ctx context.Context, filename string) error

	// GetID returns the network id of this node
	GetID() string

	// GetAddrs returns the addresses bound to this node
	GetAddrs() []string
}

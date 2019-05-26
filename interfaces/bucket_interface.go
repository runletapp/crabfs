package interfaces

import (
	"context"
	"io"
	"time"
)

// Bucket bucket managemente interface
type Bucket interface {
	// Get opens a file stream
	Get(ctx context.Context, filename string) (Fetcher, error)

	// Put writes a file to the storage
	Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error

	// Remove deletes a file from the storage
	Remove(ctx context.Context, filename string) error
}

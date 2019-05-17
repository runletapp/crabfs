package interfaces

import (
	"context"
	"io"
)

// Fetcher block fetcher interface
type Fetcher interface {
	io.Reader
	io.Seeker

	// Size returns the total size of the block map to fetch
	Size() int64
}

// FetcherFactory fetcher factory type
type FetcherFactory func(ctx context.Context, fs Core, blockMap BlockMap) (Fetcher, error)

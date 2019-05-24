package interfaces

import (
	"context"
	"io"
)

// Fetcher block fetcher interface
type Fetcher interface {
	io.Reader
	io.Seeker
	io.Closer

	// Size returns the total size of the block map to fetch
	Size() int64

	// Context returns the context of this fetcher
	// context is cancelled with the parent context or when Close is called
	Context() context.Context
}

// FetcherFactory fetcher factory type
type FetcherFactory func(ctx context.Context, fs Core, blockMap BlockMap) (Fetcher, error)

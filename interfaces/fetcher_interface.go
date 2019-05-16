package interfaces

import "io"

// Fetcher block fetcher interface
type Fetcher interface {
	io.Reader
	io.Seeker

	// Size returns the total size of the block map to fetch
	Size() int64
}

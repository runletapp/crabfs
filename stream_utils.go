package crabfs

import (
	"context"
	"io"
)

// StreamSlice represents a stream that will resolve to a slice of data determined by Offset and Length
type StreamSlice struct {
	io.Reader

	Offset int64
	Length int64
}

// StreamContext holds the context and cancel func of a stream operation
type StreamContext struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

package interfaces

import (
	"context"
	"io"
	"time"
)

type Bucket interface {
	Address() string

	Put(ctx context.Context, filename string, r io.Reader, mtime time.Time) error
	Get(ctx context.Context, filename string) (Fetcher, error)

	Remove(ctx context.Context, filename string) error
}

package interfaces

import (
	"context"
	"io"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	pb "github.com/runletapp/crabfs/protos"
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
type FetcherFactory func(ctx context.Context, fs Core, object *pb.CrabObject, privateKey crabfsCrypto.PrivKey) (Fetcher, error)

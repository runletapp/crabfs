package interfaces

import (
	"context"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
)

// Core interface
type Core interface {
	// GetID returns the network id of this node
	GetID() string

	// GetAddrs returns the addresses bound to this node
	GetAddrs() []string

	// Host returns the currently used host
	Host() Host

	// Close closes this instance and stop all children goroutines
	Close() error

	// WithBucketFiles wraps calls to a single files bucket
	WithBucketFiles(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketAddr string) (Bucket, error)

	// WithBucketFilesRoot wraps calls to a single bucket and base directory
	WithBucketFilesRoot(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketAddr string, baseDir string) (Bucket, error)
}

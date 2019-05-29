package interfaces

import (
	"context"
	"io"
	"time"

	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/identity"
)

// Core interface
type Core interface {
	// Get opens a file stream
	Get(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (Fetcher, error)

	// Put writes a file to the storage
	Put(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, file io.Reader, mtime time.Time) error

	// Remove deletes a file from the storage
	Remove(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) error

	// GetID returns the network id of this node
	GetID() string

	// GetAddrs returns the addresses bound to this node
	GetAddrs() []string

	// Blockstore returns the currently used blockstore
	Blockstore() ipfsBlockstore.Blockstore

	// Host returns the currently used host
	Host() Host

	// GarbageCollector returns the garbage collector associated with this instance
	GarbageCollector() GarbageCollector

	// Close closes this instance and stop all children goroutines
	Close() error

	// WithBucket wraps calls to a single bucket
	WithBucket(privateKey crabfsCrypto.PrivKey, bucket string) (Bucket, error)

	// PublishPublicKey publishes a public key to the network
	PublishPublicKey(publicKey crabfsCrypto.PubKey) error

	// GetIdentity returns the current identity of the node
	GetIdentity() identity.Identity
}

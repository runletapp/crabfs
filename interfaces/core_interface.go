package interfaces

import (
	"context"
	"io"
	"time"

	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/identity"
	pb "github.com/runletapp/crabfs/protos"
)

// Core interface
type Core interface {
	// Get opens a file stream
	Get(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (Fetcher, error)

	// Put writes a file to the storage
	Put(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, file io.Reader, mtime time.Time) error

	// PutAndLock writes a file to the storage and locks it
	PutAndLock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, file io.Reader, mtime time.Time) (*pb.LockToken, error)

	// Remove deletes a file from the storage
	Remove(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) error

	// Lock locks a file to avoid replublishing and overwritting during sequential updates from a single writer
	Lock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (*pb.LockToken, error)

	// Unlock unlocks a file. See Lock
	Unlock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, token *pb.LockToken) error

	// IsLocked check if a file is locked
	IsLocked(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (bool, error)

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

	// WithBucketRoot wraps calls to a single bucket and base directory
	WithBucketRoot(privateKey crabfsCrypto.PrivKey, bucket string, baseDir string) (Bucket, error)

	// PublishPublicKey publishes a public key to the network
	PublishPublicKey(publicKey crabfsCrypto.PubKey) error

	// GetIdentity returns the current identity of the node
	GetIdentity() identity.Identity
}

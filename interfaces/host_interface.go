package interfaces

import (
	"context"
	"io"
	"time"

	cid "github.com/ipfs/go-cid"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	pb "github.com/runletapp/crabfs/protos"

	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
)

// Host p2p host abstraction
type Host interface {
	// Close closes this host and all open connections
	Close() error

	// Announce
	Announce() error

	// GetSwarmPublicKey get the swarm public key from the keystore
	GetSwarmPublicKey(ctx context.Context, hash string) (crabfsCrypto.PubKey, error)

	// Publish publishes a block map
	Publish(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap BlockMap, mtime time.Time, size int64) error

	// PublishAndLock publishes a locked block map
	PublishAndLock(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap BlockMap, mtime time.Time, size int64) (*pb.LockToken, error)

	// PublishWithCacheTTL same as Publish but specifies a cache ttl timeout.
	// This will cache the content in the nodes, no updates will be retrieved
	// from the network until the ttl is timed out
	// ttl is in seconds.
	PublishWithCacheTTL(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap BlockMap, mtime time.Time, size int64, ttl uint64) error

	// Remove removes content from the network
	Remove(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) error

	// GetContent get the block map specified by 'filename
	GetContent(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (*pb.CrabObject, error)

	// Lock locks a file to avoid replublishing and overwritting during sequential updates from a single writer
	Lock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (*pb.LockToken, error)

	// Unlock unlocks a file. See Lock
	Unlock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, token *pb.LockToken) error

	// IsLocked check if a file is locked
	IsLocked(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (bool, error)

	// FindProviders find the closest providers of cid
	FindProviders(ctx context.Context, blockMeta *pb.BlockMetadata) <-chan libp2pPeerstore.PeerInfo

	// CreateBlockStream downloads a block 'cid' from peer
	CreateBlockStream(ctx context.Context, blockMeta *pb.BlockMetadata, peer *libp2pPeerstore.PeerInfo) (io.Reader, error)

	// GetID returns the network id of this host
	GetID() string

	// GetAddrs returns the addresses bound to this host
	GetAddrs() []string

	// Reprovide republish blocks and block metas to the network
	Reprovide(ctx context.Context, withBlocks bool) error

	// PutPublicKey broadcast this public key to the network
	PutPublicKey(publicKey crabfsCrypto.PubKey) error

	// Provide publishes a block in the network
	Provide(ctx context.Context, cid cid.Cid) error
}

package interfaces

import (
	"context"
	"io"

	blocks "github.com/ipfs/go-block-format"
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

	BucketAppend(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketRoot string, entry *pb.CrabEntry) error

	// CreateBlockStream downloads a block 'cid' from peer
	CreateBlockStream(ctx context.Context, blockMeta *pb.BlockMetadata, peer *libp2pPeerstore.PeerInfo) (io.Reader, error)

	// GetID returns the network id of this host
	GetID() string

	// GetAddrs returns the addresses bound to this host
	GetAddrs() []string

	PutBlock(ctx context.Context, block blocks.Block) (cid.Cid, error)

	GetBlock(ctx context.Context, cid cid.Cid) (io.Reader, error)

	CreateBucket(ctx context.Context, name string, privateKey crabfsCrypto.PrivKey) (string, *pb.CrabBucket, error)

	VerifyBucketSignature(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketAddr string) error

	SyncBucket(ctx context.Context, bucketAddr string) error

	GetBucketBook(ctx context.Context, bucketAddr string) (EntryBook, error)
}

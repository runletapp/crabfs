package interfaces

import (
	"context"
	"io"
	"time"

	pb "github.com/runletapp/crabfs/protos"

	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
)

// Host p2p host abstraction
type Host interface {
	// Announce
	Announce() error

	// GetSwarmPublicKey get the swarm public key from the keystore
	GetSwarmPublicKey(ctx context.Context, hash string) (*libp2pCrypto.RsaPublicKey, error)

	// Publish publishes a block map
	Publish(ctx context.Context, filename string, blockMap BlockMap, mtime time.Time, size int64) error

	// GetContent get the block map specified by 'filename
	GetContent(ctx context.Context, filename string) (BlockMap, error)

	// FindProviders find the closest providers of cid
	FindProviders(ctx context.Context, blockMeta *pb.BlockMetadata) <-chan libp2pPeerstore.PeerInfo

	// CreateBlockStream downloads a block 'cid' from peer
	CreateBlockStream(ctx context.Context, blockMeta *pb.BlockMetadata, peer *libp2pPeerstore.PeerInfo) (io.Reader, error)

	// GetID returns the network id of this host
	GetID() string

	// GetAddrs returns the addresses bound to this host
	GetAddrs() []string
}

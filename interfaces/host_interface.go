package interfaces

import (
	"context"
	"io"
	"time"

	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"

	ipfsDatastore "github.com/ipfs/go-datastore"
	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
)

// Host p2p host abstraction
type Host interface {
	// Announce
	Announce() error

	// GetSwarmPublicKey get the swarm public key from the keystore
	GetSwarmPublicKey(ctx context.Context, hash string) (PubKey, error)

	// Publish publishes a block map
	Publish(ctx context.Context, privateKey PrivKey, bucket string, filename string, blockMap BlockMap, mtime time.Time, size int64) error

	// Remove removes content from the network
	Remove(ctx context.Context, privateKey PrivKey, bucket string, filename string) error

	// GetContent get the block map specified by 'filename
	GetContent(ctx context.Context, publicKey PubKey, bucket string, filename string) (BlockMap, error)

	// FindProviders find the closest providers of cid
	FindProviders(ctx context.Context, blockMeta *pb.BlockMetadata) <-chan libp2pPeerstore.PeerInfo

	// CreateBlockStream downloads a block 'cid' from peer
	CreateBlockStream(ctx context.Context, blockMeta *pb.BlockMetadata, peer *libp2pPeerstore.PeerInfo) (io.Reader, error)

	// GetID returns the network id of this host
	GetID() string

	// GetAddrs returns the addresses bound to this host
	GetAddrs() []string

	// Reprovide republish blocks and block metas to the network
	Reprovide(ctx context.Context) error

	// PutPublicKey broadcast this public key to the network
	PutPublicKey(publicKey PubKey) error
}

// HostFactory type
type HostFactory func(settings *options.Settings, privateKey *libp2pCrypto.RsaPrivateKey, ds ipfsDatastore.Batching, blockstore blockstore.Blockstore) (Host, error)

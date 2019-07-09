package crabfs

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/ipfs/go-cid"

	"github.com/google/uuid"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"

	ipfsLog "berty.tech/go-ipfs-log"
	"berty.tech/go-ipfs-log/identityprovider"
	"berty.tech/go-ipfs-log/keystore"
	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreSync "github.com/ipfs/go-datastore/sync"
	ipfsConfig "github.com/ipfs/go-ipfs-config"
	ipfsCore "github.com/ipfs/go-ipfs/core"
	ipfsCoreApi "github.com/ipfs/go-ipfs/core/coreapi"
	ipfsP2P "github.com/ipfs/go-ipfs/core/node/libp2p"
	ipfsPluginLoader "github.com/ipfs/go-ipfs/plugin/loader"
	ipfsRepo "github.com/ipfs/go-ipfs/repo"
	ipfsFSRepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	ipfsCoreApiIface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/multiformats/go-multiaddr"
)

var _ interfaces.Core = &crabFS{}

type crabFS struct {
	settings *options.Settings

	ks       keystore.Interface
	identity *identityprovider.Identity

	node *ipfsCore.IpfsNode
	api  ipfsCoreApiIface.CoreAPI
	repo ipfsRepo.Repo
}

// New create a new CrabFS
func New(opts ...options.Option) (interfaces.Core, error) {
	settings := options.Settings{}
	if err := settings.SetDefaults(); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(&settings); err != nil {
			return nil, err
		}
	}

	ds := ipfsDatastoreSync.MutexWrap(ipfsDatastore.NewMapDatastore())
	ks, err := keystore.NewKeystore(ds)
	if err != nil {
		return nil, err
	}

	identity, err := identityprovider.CreateIdentity(&identityprovider.CreateIdentityOptions{
		Keystore: ks,
		ID:       settings.ID,
		Type:     "orbitdb",
	})
	if err != nil {
		return nil, err
	}

	plugins, err := ipfsPluginLoader.NewPluginLoader(path.Join(settings.Root, "plugins"))
	if err != nil {
		return nil, err
	}
	if err := plugins.Initialize(); err != nil {
		return nil, err
	}
	if err := plugins.Inject(); err != nil {
		return nil, err
	}

	if !ipfsFSRepo.IsInitialized(settings.Root) {
		config, err := ipfsConfig.Init(os.Stdout, 2048)
		if err != nil {
			return nil, err
		}

		config.Addresses.Swarm = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", settings.Port),
			fmt.Sprintf("/ip6/::/tcp/%d", settings.Port),
		}
		config.Swarm.DisableNatPortMap = true
		config.Swarm.ConnMgr.Type = "basic"
		config.Swarm.ConnMgr.HighWater = 10
		config.Swarm.ConnMgr.LowWater = 1

		bpaddrs, err := ipfsConfig.ParseBootstrapPeers(settings.BootstrapPeers)
		if err != nil {
			return nil, err
		}
		config.SetBootstrapPeers(bpaddrs)

		config.Discovery.MDNS.Enabled = true

		if err := ipfsFSRepo.Init(settings.Root, config); err != nil {
			return nil, err
		}
	}

	repo, err := ipfsFSRepo.Open(settings.Root)
	if err != nil {
		return nil, err
	}

	cfg := &ipfsCore.BuildCfg{
		Repo:    repo,
		Online:  true,
		Routing: ipfsP2P.DHTClientOption,
		// Permanent: true,
	}

	node, err := ipfsCore.NewNode(settings.Context, cfg)
	if err != nil {
		return nil, err
	}

	api, err := ipfsCoreApi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}

	fs := &crabFS{
		settings: &settings,

		ks:       ks,
		identity: identity,

		repo: repo,
		node: node,
		api:  api,
	}

	return fs, nil
}

func (fs *crabFS) Close() error {
	if err := fs.node.Close(); err != nil {
		return err
	}

	return fs.repo.Close()
}

func (fs *crabFS) GetID() string {
	return fs.node.PeerHost.ID().Pretty()
}

func (fs *crabFS) GetAddrs() []string {
	addrs := []string{}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", fs.node.PeerHost.ID().Pretty()))
	if err != nil {
		return addrs
	}

	for _, addr := range fs.node.PeerHost.Addrs() {
		addrs = append(addrs, addr.Encapsulate(hostAddr).String())
	}

	return addrs
}

func (fs *crabFS) Create(ctx context.Context, privKey crabfsCrypto.PrivKey) (interfaces.Bucket, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	log, err := ipfsLog.NewLog(fs.api, fs.identity, &ipfsLog.LogOptions{ID: id.String()})
	if err != nil {
		return nil, err
	}

	rootData, err := NewBucketRoot(ctx, id.String())
	if err != nil {
		return nil, err
	}

	if _, err := log.Append(ctx, rootData, 1); err != nil {
		return nil, err
	}

	cid, err := log.ToMultihash(ctx)
	if err != nil {
		return nil, err
	}

	return NewBucket(fs, log, cid.String(), privKey)
}

func (fs *crabFS) Open(ctx context.Context, addr string, privKey crabfsCrypto.PrivKey) (interfaces.Bucket, error) {
	cid, err := cid.Decode(addr)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Opening addr\n")
	log, err := ipfsLog.NewFromMultihash(ctx, fs.api, fs.identity, cid, &ipfsLog.LogOptions{}, &ipfsLog.FetchOptions{})
	if err != nil {
		return nil, err
	}
	fmt.Printf("Opening addr DONE\n")

	return NewBucket(fs, log, addr, privKey)
}

func (fs *crabFS) Storage() ipfsCoreApiIface.CoreAPI {
	return fs.api
}

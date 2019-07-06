package crabfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"

	ipfsConfig "github.com/ipfs/go-ipfs-config"
	ipfsCore "github.com/ipfs/go-ipfs/core"
	ipfsCoreApi "github.com/ipfs/go-ipfs/core/coreapi"
	ipfsPluginLoader "github.com/ipfs/go-ipfs/plugin/loader"
	ipfsRepo "github.com/ipfs/go-ipfs/repo"
	ipfsFSRepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	ipfsCoreApiIface "github.com/ipfs/interface-go-ipfs-core"
	ipfsPath "github.com/ipfs/interface-go-ipfs-core/path"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
	cache "github.com/patrickmn/go-cache"
)

var _ interfaces.Host = &hostImpl{}

const (
	// ProtocolV1 crabfs version 1
	ProtocolV1 = "/crabfs/v1"
)

type hostImpl struct {
	settings *options.Settings

	bucketBooks      map[string]interfaces.EntryBook
	bucketBooksMutex sync.RWMutex

	bucketCache *cache.Cache

	node *ipfsCore.IpfsNode
	api  ipfsCoreApiIface.CoreAPI
	repo ipfsRepo.Repo
}

// HostNew creates a new host
func HostNew(settings *options.Settings) (interfaces.Host, error) {
	newHost := &hostImpl{
		settings: settings,

		bucketBooks:      map[string]interfaces.EntryBook{},
		bucketBooksMutex: sync.RWMutex{},

		bucketCache: cache.New(5*time.Minute, 5*time.Minute),
	}

	ipfsRoot := path.Join(settings.Root, "ipfs")

	plugins, err := ipfsPluginLoader.NewPluginLoader(path.Join(ipfsRoot, "plugins"))
	if err != nil {
		return nil, err
	}
	if err := plugins.Initialize(); err != nil {
		return nil, err
	}
	if err := plugins.Inject(); err != nil {
		return nil, err
	}

	if !ipfsFSRepo.IsInitialized(ipfsRoot) {
		config, err := ipfsConfig.Init(os.Stdout, 2048)
		if err != nil {
			return nil, err
		}

		config.Addresses.Swarm = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", settings.Port),
			fmt.Sprintf("/ip6/::/tcp/%d", settings.Port),
		}
		config.Swarm.DisableNatPortMap = true
		// config.Swarm.ConnMgr.Type = "basic"
		// config.Swarm.ConnMgr.HighWater = 10
		// config.Swarm.ConnMgr.LowWater = 1

		bpaddrs, err := ipfsConfig.ParseBootstrapPeers(settings.BootstrapPeers)
		if err != nil {
			return nil, err
		}
		config.SetBootstrapPeers(bpaddrs)

		config.Discovery.MDNS.Enabled = false

		if err := ipfsFSRepo.Init(ipfsRoot, config); err != nil {
			return nil, err
		}
	}

	repo, err := ipfsFSRepo.Open(ipfsRoot)
	if err != nil {
		return nil, err
	}

	cfg := &ipfsCore.BuildCfg{
		Repo:   repo,
		Online: true,
		// Permanent: true,
		// Routing: dhtCreator,
	}

	node, err := ipfsCore.NewNode(settings.Context, cfg)
	if err != nil {
		return nil, err
	}

	api, err := ipfsCoreApi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}

	newHost.node = node
	newHost.api = api
	newHost.repo = repo

	return newHost, nil
}

func (host *hostImpl) Close() error {
	if err := host.node.Close(); err != nil {
		return err
	}

	if err := host.repo.Close(); err != nil {
		return err
	}

	return nil
}

func (host *hostImpl) Announce() error {
	// routingDiscovery := discovery.NewRoutingDiscovery(host.dht)

	// // Force first advertise
	// routingDiscovery.Advertise(host.settings.Context, "crabfs")
	// // for {
	// // 	_, err := routingDiscovery.Advertise(host.settings.Context, "crabfs")
	// // 	if err == nil {
	// // 		break
	// // 	}
	// // 	<-time.After(1 * time.Second)
	// // 	// log.Printf("Err here: %v", err)
	// // }

	// discovery.Advertise(host.settings.Context, routingDiscovery, "crabfs")

	// peers, err := discovery.FindPeers(host.settings.Context, routingDiscovery, "crabfs")
	// if err != nil {
	// 	log.Printf("Discovery err: %v", err)
	// } else {
	// 	log.Printf("Found %d peers", len(peers))
	// }

	// if host.dht.RoutingTable().Size() == 0 {
	// 	panic("Routing table 0")
	// }

	// ch, err := routingDiscovery.FindPeers(host.settings.Context, "crabfs", discoveryOptions.Limit(5))
	// if err != nil {
	// 	return err
	// }
	// for peer := range ch {
	// 	log.Printf("Connecting to: %v %v", peer.String(), peer.Addrs)
	// 	if err := host.p2pHost.Connect(host.settings.Context, peer); err != nil {
	// 		log.Printf("Err: %v", err)
	// 	}
	// }

	return nil
}

func (host *hostImpl) PutBlock(ctx context.Context, block blocks.Block) (cid.Cid, error) {
	stat, err := host.api.Block().Put(ctx, bytes.NewReader(block.RawData()))
	if err != nil {
		return cid.Cid{}, err
	}

	return stat.Path().Cid(), nil
}

func (host *hostImpl) GetID() string {
	return host.node.PeerHost.ID().Pretty()
}

func (host *hostImpl) GetAddrs() []string {
	addrs := []string{}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.node.PeerHost.ID().Pretty()))
	if err != nil {
		return addrs
	}

	for _, addr := range host.node.PeerHost.Addrs() {
		addrs = append(addrs, addr.Encapsulate(hostAddr).String())
	}

	return addrs
}

func (host *hostImpl) GetSwarmPublicKey(ctx context.Context, hash string) (crabfsCrypto.PubKey, error) {
	return nil, nil
	// data, err := host.dht.GetValue(ctx, fmt.Sprintf("/crabfs_pk/%s", hash))
	// if err != nil {
	// 	return nil, err
	// }

	// pk, err := crabfsCrypto.UnmarshalPublicKey(data)
	// if err != nil {
	// 	return nil, err
	// }

	// return pk, nil
}

func (host *hostImpl) CreateBucket(ctx context.Context, name string, privateKey crabfsCrypto.PrivKey) (string, *pb.CrabBucket, error) {
	bucket := &pb.CrabBucket{
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Name:      name,
	}

	bucketData, err := proto.Marshal(bucket)
	if err != nil {
		return "", nil, err
	}

	signature, err := privateKey.Sign(bucketData)
	if err != nil {
		return "", nil, err
	}

	bucket.Signature = signature

	bucketJSON, err := proto.Marshal(bucket)
	if err != nil {
		return "", nil, err
	}

	nd, err := host.api.Object().New(ctx)
	if err != nil {
		return "", nil, err
	}

	resolved, err := host.api.Object().SetData(ctx, ipfsPath.IpfsPath(nd.Cid()), bytes.NewReader(bucketJSON))
	if err != nil {
		return "", nil, err
	}

	entries, err := EntryBookNew()
	if err != nil {
		return "", nil, err
	}

	bucketRoot := resolved.Cid().String()

	host.bucketBooksMutex.Lock()
	host.bucketBooks[bucketRoot] = entries
	host.bucketBooksMutex.Unlock()

	host.bucketCache.SetDefault(bucketRoot, bucket)

	return bucketRoot, bucket, nil
}

func (host *hostImpl) BucketAppend(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketRoot string, entry *pb.CrabEntry) error {
	entryData, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	signature, err := privateKey.Sign(entryData)
	if err != nil {
		return err
	}

	entry.Signature = signature

	entryJSON, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	nd, err := host.api.Object().New(ctx)
	if err != nil {
		return err
	}

	resolved, err := host.api.Object().AppendData(ctx, ipfsPath.IpfsPath(nd.Cid()), bytes.NewReader(entryJSON))
	if err != nil {
		return err
	}

	resolved, err = host.api.Object().AddLink(ctx, resolved, "root", ipfsPath.New(bucketRoot))
	if err != nil {
		return err
	}

	host.bucketBooksMutex.RLock()
	defer host.bucketBooksMutex.RUnlock()

	book := host.bucketBooks[bucketRoot]
	return book.Add(resolved.Cid(), entry)
}

func (host *hostImpl) get(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (*pb.CrabEntry, *pb.CrabObject, error) {
	return nil, nil, nil
	// bucketFilename := path.Join(bucket, filename)
	// publicKeyHash := publicKey.HashString()

	// key := KeyFromFilename(publicKeyHash, bucketFilename)
	// cacheKey := CacheKeyFromFilename(publicKeyHash, bucketFilename)

	// cache := host.getCache(ctx, cacheKey)

	// var object *pb.CrabObject
	// var record pb.DHTNameRecord

	// dataLocal, err := host.ds.Get(ipfsDatastore.NewKey(key))
	// if err == nil {
	// 	if err := proto.Unmarshal(dataLocal, &record); err == nil {
	// 		object, err = host.xObjectFromRecord(&record)
	// 		if err == nil {
	// 			// Early return. The object is cached and the ttl is still valid
	// 			if cache != nil && (time.Duration(object.CacheTTL)*time.Second) > time.Now().Sub(*cache) {
	// 				return &record, object, nil
	// 			}
	// 		}
	// 	}

	// }

	// // Not found in local query and/or cache is outdated. Query remote
	// data, err := host.dht.GetValue(ctx, key)

	// if err != nil && (err == libp2pRouting.ErrNotFound || strings.Contains(err.Error(), "failed to find any peer in table")) {
	// 	if object == nil {
	// 		return nil, nil, ErrObjectNotFound
	// 	}

	// 	return &record, object, nil
	// } else if err != nil {
	// 	return nil, nil, err
	// }

	// if err := proto.Unmarshal(data, &record); err != nil {
	// 	return nil, nil, err
	// }

	// object, err = host.xObjectFromRecord(&record)
	// if err != nil {
	// 	return nil, nil, err
	// }

	// // Only republish if there's no lock
	// if !host.isObjectLocked(object) {
	// 	if err := host.ds.Put(ipfsDatastore.NewKey(key), data); err != nil {
	// 		// Log
	// 	}
	// }

	// if err := host.ds.Put(ipfsDatastore.NewKey(cacheKey), []byte(time.Now().UTC().Format(time.RFC3339Nano))); err != nil {
	// 	return nil, nil, err
	// }

	// return &record, object, nil
}

func (host *hostImpl) GetContent(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (*pb.CrabObject, error) {
	_, object, err := host.get(ctx, publicKey, bucket, filename)
	return object, err
}

func (host *hostImpl) generateLockToken() (*pb.LockToken, error) {
	tk, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &pb.LockToken{
		Token: tk.String(),
	}, nil
}

func (host *hostImpl) isObjectLocked(object *pb.CrabObject) bool {
	return object.Lock != nil
}

func (host *hostImpl) IsLocked(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (bool, error) {
	_, object, err := host.get(ctx, publicKey, bucket, filename)
	if err != nil {
		return false, err
	}

	// TODO: check if the lock owner is still alive
	return host.isObjectLocked(object), nil
}

func (host *hostImpl) CreateBlockStream(ctx context.Context, blockMeta *pb.BlockMetadata, peer *libp2pPeerstore.PeerInfo) (io.Reader, error) {
	return nil, fmt.Errorf("TODO")
	// circuitAddr, err := multiaddr.NewMultiaddr("/p2p-circuit/p2p/" + peer.ID.Pretty())
	// if err != nil {
	// 	return nil, err
	// }

	// // Add the relay addr circuit to this peer
	// host.p2pHost.Peerstore().AddAddr(peer.ID, circuitAddr, libp2pPeerstore.TempAddrTTL)

	// host.p2pHost.Network().(*libp2pSwarm.Swarm).Backoff().Clear(peer.ID)
	// stream, err := host.p2pHost.NewStream(ctx, peer.ID, ProtocolV1)
	// if err != nil {
	// 	return nil, err
	// }

	// request := pb.BlockStreamRequest{
	// 	Cid: blockMeta.Cid,
	// }

	// data, err := proto.Marshal(&request)
	// if err != nil {
	// 	return nil, err
	// }

	// _, err = stream.Write(data)
	// if err != nil {
	// 	return nil, err
	// }

	// return stream, stream.Close()
}

func (host *hostImpl) getIpldData(ctx context.Context, path ipfsPath.Path) (io.Reader, error) {
	node, err := host.api.Object().Get(ctx, path)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(node.RawData()), nil
}

func (host *hostImpl) VerifyBucketSignature(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketAddr string) error {
	var bucket pb.CrabBucket

	bucketI, prs := host.bucketCache.Get(bucketAddr)
	if !prs {
		r, err := host.getIpldData(ctx, ipfsPath.New(bucketAddr))
		if err != nil {
			return err
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		if err := proto.Unmarshal(data, &bucket); err != nil {
			return err
		}
	} else {
		bucket = *(bucketI.(*pb.CrabBucket))
	}

	host.bucketCache.SetDefault(bucketAddr, &bucket)

	verifyBucket := pb.CrabBucket{
		CreatedAt: bucket.CreatedAt,
		Name:      bucket.Name,
	}

	data, err := proto.Marshal(&verifyBucket)
	if err != nil {
		return err
	}

	v, err := privateKey.GetPublic().Verify(data, bucket.Signature)
	if err != nil {
		return err
	}

	if !v {
		return ErrBucketInvalidPrivateKey
	}

	return nil
}

func (host *hostImpl) SyncBucket(ctx context.Context, bucketAddr string) error {
	getOrCreateBook := func() (interfaces.EntryBook, error) {
		host.bucketBooksMutex.Lock()
		defer host.bucketBooksMutex.Unlock()

		book, prs := host.bucketBooks[bucketAddr]
		if !prs {
			var err error
			book, err = EntryBookNew()
			if err != nil {
				return nil, err
			}
		}

		host.bucketBooks[bucketAddr] = book

		return book, nil
	}

	_, err := getOrCreateBook()
	if err != nil {
		return err
	}

	// TODO: use pubsub to fetch data
	// TODO: verify entries signatures

	return nil
}

func (host *hostImpl) GetBucketBook(ctx context.Context, bucketAddr string) (interfaces.EntryBook, error) {
	host.bucketBooksMutex.Lock()
	defer host.bucketBooksMutex.Unlock()

	book, prs := host.bucketBooks[bucketAddr]

	if !prs {
		return nil, ErrBucketUnkown
	}

	return book, nil
}

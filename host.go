package crabfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"

	"github.com/GustavoKatel/asyncutils/executor"
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
	libp2pNet "github.com/libp2p/go-libp2p-net"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
)

var _ interfaces.Host = &hostImpl{}

const (
	// ProtocolV1 crabfs version 1
	ProtocolV1 = "/crabfs/v1"
)

type hostImpl struct {
	settings    *options.Settings
	addressBook interfaces.AddressBook

	node *ipfsCore.IpfsNode
	api  ipfsCoreApiIface.CoreAPI
	repo ipfsRepo.Repo

	provideWorker executor.Executor
}

// HostNew creates a new host
func HostNew(settings *options.Settings, addressBook interfaces.AddressBook) (interfaces.Host, error) {
	provideWorker, err := executor.NewDefaultExecutorContext(settings.Context, 4)
	if err != nil {
		return nil, err
	}

	if err := provideWorker.Start(); err != nil {
		return nil, err
	}

	newHost := &hostImpl{
		settings:      settings,
		addressBook:   addressBook,
		provideWorker: provideWorker,
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

func (host *hostImpl) PutPublicKey(publicKey crabfsCrypto.PubKey) error {
	publicKeyData, err := publicKey.Marshal()
	if err != nil {
		return err
	}

	return host.dhtPutValue(host.settings.Context, fmt.Sprintf("/crabfs_pk/%s", publicKey.HashString()), publicKeyData)
}

func (host *hostImpl) handleStreamV1(stream libp2pNet.Stream) {
	defer stream.Close()

	// data, err := ioutil.ReadAll(stream)
	// if err != nil {
	// 	return
	// }

	// var request pb.BlockStreamRequest
	// if err := proto.Unmarshal(data, &request); err != nil {
	// 	return
	// }

	// cid, err := cid.Cast(request.Cid)
	// if err != nil {
	// 	return
	// }

	// block, err := host.blockstore.Get(cid)
	// if err != nil {
	// 	return
	// }

	// _, err = stream.Write(block.RawData())
	// if err != nil {
	// 	return
	// }
}

func (host *hostImpl) Reprovide(ctx context.Context, withBlocks bool) error {
	// query := ipfsDatastoreQuery.Query{
	// 	Prefix: "/crabfs/",
	// }

	// results, err := host.ds.Query(query)
	// if err != nil {
	// 	return err
	// }

	// for result := range results.Next() {
	// 	var record pb.DHTNameRecord
	// 	if err := proto.Unmarshal(result.Value, &record); err != nil {
	// 		// Invalid key, do not reprovide it
	// 		continue
	// 	}
	// 	object, err := host.xObjectFromRecord(&record)
	// 	if err != nil {
	// 		// Invalid key, do not reprovide it
	// 		continue
	// 	}

	// 	if host.isObjectLocked(object) {
	// 		// Do not reprovide locked objects
	// 		continue
	// 	}

	// 	host.dhtPutValue(ctx, result.Key, result.Value)
	// }

	// if withBlocks {
	// 	ch, err := host.blockstore.AllKeysChan(ctx)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	for cid := range ch {
	// 		_cid := cid
	// 		host.provideWorker.PostJob(func(ctx context.Context) error {
	// 			return host.Provide(ctx, _cid)
	// 		})
	// 	}
	// }

	return nil
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

func (host *hostImpl) dbAppend(ctx context.Context, bucket string, entry *pb.CrabEntry) error {
	return fmt.Errorf("TODO")
}

func (host *hostImpl) publishObject(ctx context.Context, privateKey crabfsCrypto.PrivKey, object *pb.CrabObject, bucket string) error {
	entry := &pb.CrabEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	data, err := proto.Marshal(object)
	if err != nil {
		return err
	}

	entry.Data = data

	entryData, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	signature, err := privateKey.Sign(entryData)
	if err != nil {
		return err
	}

	entry.Signature = signature

	return host.dbAppend(ctx, bucket, entry)
}

func (host *hostImpl) Publish(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap interfaces.BlockMap, mtime time.Time, size int64) error {
	bucketFilename := path.Join(bucket, filename)

	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	object := &pb.CrabObject{
		Name:   key,
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   size,
		Key:    cipherKey,
		Delete: false,
		Lock:   nil,
	}

	return host.publishObject(ctx, privateKey, object, bucket)
}

func (host *hostImpl) PublishAndLock(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap interfaces.BlockMap, mtime time.Time, size int64) (*pb.LockToken, error) {
	token, err := host.generateLockToken()
	if err != nil {
		return nil, err
	}

	tokenData, err := proto.Marshal(token)
	if err != nil {
		return nil, err
	}

	bucketFilename := path.Join(bucket, filename)

	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	object := &pb.CrabObject{
		Name:   key,
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   size,
		Key:    cipherKey,
		Delete: false,
		Lock:   tokenData,
	}

	if err := host.publishObject(ctx, privateKey, object, bucket); err != nil {
		return nil, err
	}

	return token, nil
}

func (host *hostImpl) PublishWithCacheTTL(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap interfaces.BlockMap, mtime time.Time, size int64, ttl uint64) error {
	bucketFilename := path.Join(bucket, filename)

	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	object := &pb.CrabObject{
		Name:     key,
		Blocks:   blockMap,
		Mtime:    mtime.UTC().Format(time.RFC3339Nano),
		Size:     size,
		Key:      cipherKey,
		Delete:   false,
		Lock:     nil,
		CacheTTL: ttl,
	}

	return host.publishObject(ctx, privateKey, object, bucket)
}

func (host *hostImpl) Provide(ctx context.Context, cid cid.Cid) error {
	// if host.dht.RoutingTable().Size() > 0 {
	// 	return host.dht.Provide(ctx, cid, true)
	// }

	return nil
}

func (host *hostImpl) dhtPutValue(ctx context.Context, key string, value []byte) error {
	// if err := host.ds.Put(ipfsDatastore.NewKey(key), value); err != nil {
	// 	return err
	// }

	// if host.dht.RoutingTable().Size() > 0 {
	// 	return host.dht.PutValue(ctx, key, value)
	// }

	return nil
}

func (host *hostImpl) xObjectFromRecord(record *pb.CrabEntry) (*pb.CrabObject, error) {
	var object pb.CrabObject
	if err := proto.Unmarshal(record.Data, &object); err != nil {
		return nil, err
	}

	return &object, nil
}

func (host *hostImpl) getCache(ctx context.Context, key string) *time.Time {
	// cacheData, err := host.ds.Get(ipfsDatastore.NewKey(key))
	// if err != nil {
	// 	return nil
	// }

	// cache, err := time.Parse(time.RFC3339Nano, string(cacheData))
	// if err != nil {
	// 	return nil
	// }

	// return &cache
	return nil
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

func (host *hostImpl) Lock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (*pb.LockToken, error) {
	_, object, err := host.get(ctx, privateKey.GetPublic(), bucket, filename)
	if err != nil {
		return nil, err
	}

	if object.Lock != nil {
		// TODO: future work, store peer id in the lock and check if it's still alive,
		// otherwise release the lock
		return nil, ErrFileLocked
	}

	token, err := host.generateLockToken()
	if err != nil {
		return nil, err
	}

	tokenData, err := proto.Marshal(token)
	if err != nil {
		return nil, err
	}

	object.Lock = tokenData

	if err := host.publishObject(ctx, privateKey, object, bucket); err != nil {
		return nil, err
	}

	return token, nil
}

func (host *hostImpl) Unlock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, token *pb.LockToken) error {
	_, object, err := host.get(ctx, privateKey.GetPublic(), bucket, filename)
	if err != nil {
		return err
	}

	if object.Lock == nil {
		return nil
	}

	var lock pb.LockToken
	if err := proto.Unmarshal(object.Lock, &lock); err != nil {
		return err
	}

	if strings.Compare(lock.Token, token.Token) != 0 {
		return ErrFileLockedNotOwned
	}

	object.Lock = nil
	return host.publishObject(ctx, privateKey, object, bucket)
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

func (host *hostImpl) Remove(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) error {
	bucketFilename := path.Join(bucket, filename)

	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	object := &pb.CrabObject{
		Name:   key,
		Blocks: map[int64]*pb.BlockMetadata{},
		Mtime:  time.Now().UTC().Format(time.RFC3339Nano),
		Size:   0,
		Key:    []byte{},
		Delete: true,
	}

	return host.publishObject(ctx, privateKey, object, bucket)
}

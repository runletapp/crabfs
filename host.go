package crabfs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"time"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/identity"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"

	scheduler "github.com/GustavoKatel/asyncutils/scheduler"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"

	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreQuery "github.com/ipfs/go-datastore/query"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/libp2p/go-libp2p"
	libp2pCircuit "github.com/libp2p/go-libp2p-circuit"
	libp2pHost "github.com/libp2p/go-libp2p-core/host"
	libp2pNet "github.com/libp2p/go-libp2p-core/network"
	libp2pRouting "github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	libp2pDht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pDhtOptions "github.com/libp2p/go-libp2p-kad-dht/opts"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2pRoutedHost "github.com/libp2p/go-libp2p/p2p/host/routed"
)

var _ interfaces.Host = &hostImpl{}

const (
	// ProtocolV1 crabfs version 1
	ProtocolV1 = "/crabfs/v1"
)

type hostImpl struct {
	p2pHost libp2pHost.Host
	dht     *libp2pDht.IpfsDHT

	ds ipfsDatastore.Batching

	settings *options.Settings

	blockstore blockstore.Blockstore

	provideWorker scheduler.Scheduler
}

// HostNew creates a new host
func HostNew(settings *options.Settings, ds ipfsDatastore.Batching, blockstore blockstore.Blockstore) (interfaces.Host, error) {
	sourceMultiAddrIP4, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", settings.Port))
	if err != nil {
		return nil, err
	}

	sourceMultiAddrIP6, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", settings.Port))
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(sourceMultiAddrIP4, sourceMultiAddrIP6),
		libp2p.EnableRelay(libp2pCircuit.OptDiscovery),
	}

	id, ok := settings.Identity.(*identity.Libp2pIdentity)
	if !ok {
		return nil, fmt.Errorf("Invalid identity")
	}

	opts = append(opts, libp2p.Identity(id.GetLibp2pPrivateKey()))

	p2pHost, err := libp2p.New(
		settings.Context,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return HostNewWithP2P(settings, p2pHost, ds, blockstore)
}

// HostNewWithP2P creates a new host with an underlying p2p host
func HostNewWithP2P(settings *options.Settings, p2pHost libp2pHost.Host, ds ipfsDatastore.Batching, blockstore blockstore.Blockstore) (interfaces.Host, error) {
	provideWorker, err := scheduler.NewWithContext(settings.Context)
	if err != nil {
		return nil, err
	}

	if err := provideWorker.Start(); err != nil {
		return nil, err
	}

	newHost := &hostImpl{
		settings: settings,

		ds:         ds,
		blockstore: blockstore,

		provideWorker: provideWorker,
	}

	// Configure peer discovery and key validator
	dhtValidator := libp2pDhtOptions.NamespacedValidator(
		"crabfs",
		DHTNamespaceValidatorNew(settings.Context, newHost.GetSwarmPublicKey),
	)
	dhtPKValidator := libp2pDhtOptions.NamespacedValidator(
		"crabfs_pk",
		DHTNamespacePKValidatorNew(),
	)
	dht, err := libp2pDht.New(settings.Context, p2pHost, dhtValidator, dhtPKValidator, libp2pDhtOptions.Datastore(ds))
	if err != nil {
		return nil, err
	}

	newHost.dht = dht

	if err = dht.Bootstrap(settings.Context); err != nil {
		return nil, err
	}

	newHost.p2pHost = libp2pRoutedHost.Wrap(p2pHost, dht)

	newHost.p2pHost.SetStreamHandler(ProtocolV1, newHost.handleStreamV1)

	for _, addr := range settings.BootstrapPeers {
		newHost.connectToPeer(addr)
	}

	return newHost, nil
}

func (host *hostImpl) Announce() error {
	routingDiscovery := discovery.NewRoutingDiscovery(host.dht)
	discovery.Advertise(host.settings.Context, routingDiscovery, "crabfs")

	return nil
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

	data, err := ioutil.ReadAll(stream)
	if err != nil {
		return
	}

	var request pb.BlockStreamRequest
	if err := proto.Unmarshal(data, &request); err != nil {
		return
	}

	cid, err := cid.Cast(request.Cid)
	if err != nil {
		return
	}

	block, err := host.blockstore.Get(cid)
	if err != nil {
		return
	}

	_, err = stream.Write(block.RawData())
	if err != nil {
		return
	}
}

func (host *hostImpl) Reprovide(ctx context.Context, withBlocks bool) error {
	query := ipfsDatastoreQuery.Query{
		Prefix: "/crabfs/",
	}

	results, err := host.ds.Query(query)
	if err != nil {
		return err
	}

	for result := range results.Next() {
		var record pb.DHTNameRecord
		if err := proto.Unmarshal(result.Value, &record); err != nil {
			// Invalid key, do not reprovide it
			continue
		}
		object, err := host.xObjectFromRecord(&record)
		if err != nil {
			// Invalid key, do not reprovide it
			continue
		}

		if host.isObjectLocked(object) {
			// Do not reprovide locked objects
			continue
		}

		host.dhtPutValue(ctx, result.Key, result.Value)
	}

	if withBlocks {
		ch, err := host.blockstore.AllKeysChan(ctx)
		if err != nil {
			return err
		}

		for cid := range ch {
			host.provide(ctx, cid)
		}
	}

	return nil
}

func (host *hostImpl) connectToPeer(addr string) error {
	ma := multiaddr.StringCast(addr)

	peerinfo, err := libp2pPeerstore.InfoFromP2pAddr(ma)
	if err != nil {
		return err
	}

	return host.p2pHost.Connect(host.settings.Context, *peerinfo)
}

func (host *hostImpl) GetID() string {
	return host.p2pHost.ID().Pretty()
}

func (host *hostImpl) GetAddrs() []string {
	addrs := []string{}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.p2pHost.ID().Pretty()))
	if err != nil {
		return addrs
	}

	for _, addr := range host.p2pHost.Addrs() {
		addrs = append(addrs, addr.Encapsulate(hostAddr).String())
	}

	return addrs
}

func (host *hostImpl) GetSwarmPublicKey(ctx context.Context, hash string) (crabfsCrypto.PubKey, error) {
	data, err := host.dht.GetValue(ctx, fmt.Sprintf("/crabfs_pk/%s", hash))
	if err != nil {
		return nil, err
	}

	pk, err := crabfsCrypto.UnmarshalPublicKey(data)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

func (host *hostImpl) publishObject(ctx context.Context, privateKey crabfsCrypto.PrivKey, object *pb.CrabObject, bucket string, filename string) error {
	record := &pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	data, err := proto.Marshal(object)
	if err != nil {
		return err
	}

	record.Data = data

	signature, err := privateKey.Sign(data)
	if err != nil {
		return err
	}

	record.Signature = signature

	value, err := proto.Marshal(record)
	if err != nil {
		return err
	}

	for _, blockMeta := range object.Blocks {
		cid, _ := cid.Cast(blockMeta.Cid)
		if err := host.provideWorker.PostJob(func(ctx context.Context) error {
			return host.provide(ctx, cid)
		}); err != nil {
			return err
		}
	}

	bucketFilename := path.Join(bucket, filename)

	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	return host.dhtPutValue(ctx, key, value)
}

func (host *hostImpl) Publish(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap interfaces.BlockMap, mtime time.Time, size int64) error {
	object := &pb.CrabObject{
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   size,
		Key:    cipherKey,
		Delete: false,
		Lock:   nil,
	}

	return host.publishObject(ctx, privateKey, object, bucket, filename)
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

	object := &pb.CrabObject{
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   size,
		Key:    cipherKey,
		Delete: false,
		Lock:   tokenData,
	}

	if err := host.publishObject(ctx, privateKey, object, bucket, filename); err != nil {
		return nil, err
	}

	return token, nil
}

func (host *hostImpl) PublishWithCacheTTL(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap interfaces.BlockMap, mtime time.Time, size int64, ttl uint64) error {
	object := &pb.CrabObject{
		Blocks:   blockMap,
		Mtime:    mtime.UTC().Format(time.RFC3339Nano),
		Size:     size,
		Key:      cipherKey,
		Delete:   false,
		Lock:     nil,
		CacheTTL: ttl,
	}

	return host.publishObject(ctx, privateKey, object, bucket, filename)
}

func (host *hostImpl) provide(ctx context.Context, cid cid.Cid) error {
	if host.dht.RoutingTable().Size() > 0 {
		return host.dht.Provide(ctx, cid, true)
	}

	return nil
}

func (host *hostImpl) dhtPutValue(ctx context.Context, key string, value []byte) error {
	if err := host.ds.Put(ipfsDatastore.NewKey(key), value); err != nil {
		return err
	}

	if host.dht.RoutingTable().Size() > 0 {
		return host.dht.PutValue(ctx, key, value)
	}

	return nil
}

func (host *hostImpl) xObjectFromRecord(record *pb.DHTNameRecord) (*pb.CrabObject, error) {
	var object pb.CrabObject
	if err := proto.Unmarshal(record.Data, &object); err != nil {
		return nil, err
	}

	return &object, nil
}

func (host *hostImpl) getCache(ctx context.Context, key string) *time.Time {
	cacheData, err := host.ds.Get(ipfsDatastore.NewKey(key))
	if err != nil {
		return nil
	}

	cache, err := time.Parse(time.RFC3339Nano, string(cacheData))
	if err != nil {
		return nil
	}

	return &cache
}

func (host *hostImpl) get(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (*pb.DHTNameRecord, *pb.CrabObject, error) {
	bucketFilename := path.Join(bucket, filename)
	publicKeyHash := publicKey.HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)
	cacheKey := CacheKeyFromFilename(publicKeyHash, bucketFilename)

	cache := host.getCache(ctx, cacheKey)

	var object *pb.CrabObject
	var record pb.DHTNameRecord

	dataLocal, err := host.ds.Get(ipfsDatastore.NewKey(key))
	if err == nil {
		if err := proto.Unmarshal(dataLocal, &record); err == nil {
			object, err = host.xObjectFromRecord(&record)
			if err == nil {
				// Early return. The object is cached and the ttl is still valid
				if cache != nil && (time.Duration(object.CacheTTL)*time.Second) > time.Now().Sub(*cache) {
					return &record, object, nil
				}
			}
		}

	}

	// Not found in local query and/or cache is outdated. Query remote
	data, err := host.dht.GetValue(ctx, key)

	if err != nil && (err == libp2pRouting.ErrNotFound || strings.Contains(err.Error(), "failed to find any peer in table")) {
		if object == nil {
			return nil, nil, ErrObjectNotFound
		}

		return &record, object, nil
	} else if err != nil {
		return nil, nil, err
	}

	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, nil, err
	}

	object, err = host.xObjectFromRecord(&record)
	if err != nil {
		return nil, nil, err
	}

	// Only republish if there's no lock
	if !host.isObjectLocked(object) {
		if err := host.ds.Put(ipfsDatastore.NewKey(key), data); err != nil {
			// Log
		}
	}

	if err := host.ds.Put(ipfsDatastore.NewKey(cacheKey), []byte(time.Now().UTC().Format(time.RFC3339Nano))); err != nil {
		return nil, nil, err
	}

	return &record, object, nil
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

	if err := host.publishObject(ctx, privateKey, object, bucket, filename); err != nil {
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
	return host.publishObject(ctx, privateKey, object, bucket, filename)
}

func (host *hostImpl) FindProviders(ctx context.Context, blockMeta *pb.BlockMetadata) <-chan libp2pPeerstore.PeerInfo {
	cid, _ := cid.Cast(blockMeta.Cid)

	ch := host.dht.FindProvidersAsync(ctx, cid, libp2pDht.KValue)

	return ch
}

func (host *hostImpl) CreateBlockStream(ctx context.Context, blockMeta *pb.BlockMetadata, peer *libp2pPeerstore.PeerInfo) (io.Reader, error) {
	stream, err := host.p2pHost.NewStream(ctx, peer.ID, ProtocolV1)
	if err != nil {
		return nil, err
	}

	request := pb.BlockStreamRequest{
		Cid: blockMeta.Cid,
	}

	data, err := proto.Marshal(&request)
	if err != nil {
		return nil, err
	}

	_, err = stream.Write(data)
	if err != nil {
		return nil, err
	}

	return stream, stream.Close()
}

func (host *hostImpl) Remove(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) error {
	// Create a new record to replace the old one,
	// remove all blocks and set the delete flag to true
	record := &pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	recordValue := &pb.CrabObject{
		Blocks: map[int64]*pb.BlockMetadata{},
		Mtime:  time.Now().UTC().Format(time.RFC3339Nano),
		Size:   0,
		Key:    []byte{},
		Delete: true,
	}

	data, err := proto.Marshal(recordValue)
	if err != nil {
		return err
	}

	record.Data = data

	signature, err := privateKey.Sign(data)
	if err != nil {
		return err
	}

	record.Signature = signature

	value, err := proto.Marshal(record)
	if err != nil {
		return err
	}

	bucketFilename := path.Join(bucket, filename)
	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	return host.dhtPutValue(ctx, key, value)
}

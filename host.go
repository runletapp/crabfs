package crabfs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path"
	"time"

	blockstore "github.com/ipfs/go-ipfs-blockstore"

	"github.com/ipfs/go-cid"

	"github.com/golang/protobuf/proto"

	"github.com/multiformats/go-multiaddr"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/identity"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"

	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreQuery "github.com/ipfs/go-datastore/query"
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
	newHost := &hostImpl{
		settings: settings,

		ds:         ds,
		blockstore: blockstore,
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

func (host *hostImpl) Reprovide(ctx context.Context) error {
	query := ipfsDatastoreQuery.Query{
		Prefix: "/crabfs/",
	}

	results, err := host.ds.Query(query)
	if err != nil {
		return err
	}

	for result := range results.Next() {
		host.dhtPutValue(ctx, result.Key, result.Value)
	}

	ch, err := host.blockstore.AllKeysChan(ctx)
	if err != nil {
		return err
	}

	for cid := range ch {
		host.provide(ctx, cid)
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
	log.Printf("host addr: %v", hostAddr.String())
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

func (host *hostImpl) Publish(ctx context.Context, privateKey crabfsCrypto.PrivKey, cipherKey []byte, bucket string, filename string, blockMap interfaces.BlockMap, mtime time.Time, size int64) error {
	record := &pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	recordValue := &pb.CrabObject{
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   size,
		Key:    cipherKey,
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

	for _, blockMeta := range blockMap {
		cid, _ := cid.Cast(blockMeta.Cid)
		if err := host.provide(ctx, cid); err != nil {
			return err
		}
	}

	bucketFilename := path.Join(bucket, filename)

	publicKeyHash := privateKey.GetPublic().HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)

	return host.dhtPutValue(ctx, key, value)
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

func (host *hostImpl) GetContent(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (*pb.CrabObject, error) {
	bucketFilename := path.Join(bucket, filename)
	publicKeyHash := publicKey.HashString()

	key := KeyFromFilename(publicKeyHash, bucketFilename)
	data, err := host.dht.GetValue(ctx, key)

	// Not found in remote query, try local only
	if err != nil && err == libp2pRouting.ErrNotFound {
		var err error
		data, err = host.ds.Get(ipfsDatastore.NewKey(key))
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if err == nil {
		if err := host.ds.Put(ipfsDatastore.NewKey(key), data); err != nil {
			// Log
		}
	}

	var record pb.DHTNameRecord
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	var value pb.CrabObject
	if err := proto.Unmarshal(record.Data, &value); err != nil {
		return nil, err
	}

	return &value, nil
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
		Delete:    true,
	}

	recordValue := &pb.CrabObject{
		Blocks: map[int64]*pb.BlockMetadata{},
		Mtime:  time.Now().UTC().Format(time.RFC3339Nano),
		Size:   0,
		Key:    []byte{},
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

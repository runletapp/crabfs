package crabfs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/proto"

	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	libp2pCircuit "github.com/libp2p/go-libp2p-circuit"
	discovery "github.com/libp2p/go-libp2p-discovery"
	libp2pHost "github.com/libp2p/go-libp2p-host"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pdhtOptions "github.com/libp2p/go-libp2p-kad-dht/opts"
	libp2pNet "github.com/libp2p/go-libp2p-net"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2pProtocol "github.com/libp2p/go-libp2p-protocol"
	libp2pPubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pRoutedHost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
)

// Host controls the p2p host instance
type Host struct {
	ctx context.Context

	port int

	p2pHost         libp2pHost.Host
	dht             *libp2pdht.IpfsDHT
	discoveryKey    string
	broadcastRouter *libp2pPubsub.PubSub
}

// HostNew creates a new host instance
func HostNew(ctx context.Context, port int, discoveryKey string) (*Host, error) {
	host := &Host{
		ctx:          ctx,
		port:         port,
		discoveryKey: discoveryKey,
	}

	sourceMultiAddrIP4, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	sourceMultiAddrIP6, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	p2pHost, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddrIP4, sourceMultiAddrIP6),
		libp2p.EnableRelay(libp2pCircuit.OptDiscovery),
	)
	if err != nil {
		return nil, err
	}

	// Configure peer discovery and key validator
	dhtValidator := libp2pdhtOptions.NamespacedValidator(
		"crabfs",
		DHTNamespaceValidatorNew(discoveryKey),
	)
	dht, err := libp2pdht.New(ctx, p2pHost, dhtValidator)
	if err != nil {
		return nil, err
	}

	host.dht = dht

	if err = dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	host.p2pHost = libp2pRoutedHost.Wrap(p2pHost, dht)
	protocol := libp2pProtocol.ID(fmt.Sprintf("/crabfs/%s", discoveryKey))
	host.p2pHost.SetStreamHandler(protocol, host.handleStream)

	// broadcastProtocol := libp2pProtocol.ID(fmt.Sprintf("/crabfs/%s/broadcast", discoveryKey))
	// broadcast, err := libp2pPubsub.NewFloodsubWithProtocols(
	// 	ctx,
	// 	host.p2pHost,
	// 	[]libp2pProtocol.ID{broadcastProtocol},
	// 	libp2pPubsub.WithMessageSigning(true),
	// )
	// if err != nil {
	// 	return nil, err
	// }
	// host.broadcastRouter = broadcast

	// go host.listenSubscription()

	return host, nil
}

func (host *Host) handleStream(stream libp2pNet.Stream) {
	log.Printf("new stream")
}

// GetID returns the id of this p2p host
func (host *Host) GetID() string {
	return host.p2pHost.ID().Pretty()
}

// ConnectToPeer connect this node to a peer
func (host *Host) ConnectToPeer(addr string) error {
	ma := multiaddr.StringCast(addr)

	peerinfo, err := libp2pPeerstore.InfoFromP2pAddr(ma)
	if err != nil {
		return err
	}

	return host.p2pHost.Connect(host.ctx, *peerinfo)
}

// GetAddrs get the addresses that this node is bound to
func (host *Host) GetAddrs() []string {
	addrs := []string{}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.p2pHost.ID().Pretty()))
	if err != nil {
		return addrs
	}

	for _, addr := range host.p2pHost.Addrs() {
		addrs = append(addrs, addr.Encapsulate(hostAddr).String())
	}

	return addrs
}

// Announce announce this host in the network
func (host *Host) Announce() error {
	routingDiscovery := discovery.NewRoutingDiscovery(host.dht)
	discovery.Advertise(host.ctx, routingDiscovery, host.discoveryKey)

	return nil
}

func (host *Host) pathNameToKey(name string) (string, error) {
	nameHash, err := multihash.Sum([]byte(name), multihash.SHA3_256, -1)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/crabfs/%s/%s", host.discoveryKey, nameHash.String()), nil
}

// AnnounceContent announce that we can provide the content id 'contentID'
func (host *Host) AnnounceContent(name string, contentID *cid.Cid) error {
	err := host.dht.Provide(host.ctx, *contentID, true)
	if err != nil {
		return err
	}

	key, err := host.pathNameToKey(name)
	if err != nil {
		return err
	}

	record := DHTNameRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ContentID: contentID.Bytes(),
	}
	value, err := proto.Marshal(&record)
	if err != nil {
		return err
	}

	return host.dht.PutValue(host.ctx, key, value)
}

// GetContentID resolves the pathName to a content id
func (host *Host) GetContentID(ctx context.Context, name string) (*cid.Cid, error) {
	key, err := host.pathNameToKey(name)
	if err != nil {
		return nil, err
	}

	data, err := host.dht.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}

	var record DHTNameRecord
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	contentID, err := cid.Cast(record.ContentID)
	if err != nil {
		return nil, err
	}

	return &contentID, nil
}

// GetProviders resolves the content id to a list of peers that provides that content
func (host *Host) GetProviders(ctx context.Context, contentID *cid.Cid) ([][]string, error) {
	peers := [][]string{}

	peerInfoList, err := host.dht.FindProviders(ctx, *contentID)
	if err != nil {
		return nil, err
	}

	for _, peerInfo := range peerInfoList {
		addrs := peerInfo.Addrs

		peerAddrID, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peerInfo.ID.Pretty()))
		if err != nil {
			return nil, err
		}

		addrsStr := []string{}
		for _, addr := range addrs {
			addrsStr = append(addrsStr, addr.Encapsulate(peerAddrID).String())
		}

		peers = append(peers, addrsStr)
	}

	return peers, nil
}

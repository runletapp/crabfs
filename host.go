package crabfs

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	libp2pCircuit "github.com/libp2p/go-libp2p-circuit"
	libp2pHost "github.com/libp2p/go-libp2p-host"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pNet "github.com/libp2p/go-libp2p-net"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2pRoutedHost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
)

// Host controls the p2p host instance
type Host struct {
	ctx context.Context

	port int

	p2pHost      libp2pHost.Host
	dht          *libp2pdht.IpfsDHT
	discoveryKey string
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

	// Configure peer discovery
	dht, err := libp2pdht.New(ctx, p2pHost)
	if err != nil {
		return nil, err
	}
	host.dht = dht

	if err = dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	host.p2pHost = libp2pRoutedHost.Wrap(p2pHost, dht)
	host.p2pHost.SetStreamHandler("/crabfs", host.handleStream)

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

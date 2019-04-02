package crabfs

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	libp2pCircuit "github.com/libp2p/go-libp2p-circuit"
	libp2pHost "github.com/libp2p/go-libp2p-host"
	"github.com/multiformats/go-multiaddr"
)

// Relay controls a relay server
type Relay struct {
	ctx context.Context

	port int

	p2pHost libp2pHost.Host
}

// RelayNew creates a new relay instance
func RelayNew(ctx context.Context, port int) (*Relay, error) {
	sourceMultiAddrIP4, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	sourceMultiAddrIP6, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddrIP4, sourceMultiAddrIP6),
		libp2p.EnableRelay(libp2pCircuit.OptHop),
	)
	if err != nil {
		return nil, err
	}

	relay := &Relay{
		ctx:     ctx,
		port:    port,
		p2pHost: host,
	}

	return relay, nil
}

// GetAddrs get the addresses that this node is bound to
func (relay *Relay) GetAddrs() []string {
	addrs := []string{}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", relay.p2pHost.ID().Pretty()))
	if err != nil {
		return addrs
	}

	for _, addr := range relay.p2pHost.Addrs() {
		addrs = append(addrs, addr.Encapsulate(hostAddr).String())
	}

	return addrs
}

// GetID returns the id of this p2p relay
func (relay *Relay) GetID() string {
	return relay.p2pHost.ID().Pretty()
}

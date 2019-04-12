package crabfs

import (
	"context"
	"strings"

	"github.com/libp2p/go-libp2p"
	libp2pCircuit "github.com/libp2p/go-libp2p-circuit"
	libp2pHost "github.com/libp2p/go-libp2p-host"
	"github.com/runletapp/crabfs/options"
)

// Relay controls a relay server
type Relay struct {
	ctx context.Context

	port int

	p2pRelayHost libp2pHost.Host
	host         *CrabFS
}

// RelayNew creates a new relay instance
func RelayNew(ctx context.Context, port int, bootstrapPeers []string) (*Relay, error) {
	relayHost, err := libp2p.New(
		ctx,
		libp2p.EnableRelay(libp2pCircuit.OptHop),
	)
	if err != nil {
		return nil, err
	}

	addrs := bootstrapPeers
	for _, addr := range relayHost.Addrs() {
		if strings.HasPrefix(addr.String(), "tcp/127") || strings.HasPrefix(addr.String(), "tcp/::1") {
			addrs = append(addrs, addr.String())
		}
	}

	host, err := New(
		"tmp/relay",
		options.Port(port),
		options.RelayOnly(true),
		options.BootstrapPeers(addrs),
	)
	if err != nil {
		relayHost.Close()
		return nil, err
	}

	relay := &Relay{
		ctx:          ctx,
		port:         port,
		host:         host,
		p2pRelayHost: relayHost,
	}

	return relay, nil
}

// GetAddrs get the addresses that this node is bound to
func (relay *Relay) GetAddrs() []string {
	return relay.host.GetAddrs()
}

// GetHostID returns the id of this p2p relay
func (relay *Relay) GetHostID() string {
	return relay.host.GetHostID()
}

// GetRelayID returns the id of this p2p relay
func (relay *Relay) GetRelayID() string {
	return relay.p2pRelayHost.ID().Pretty()
}

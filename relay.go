package crabfs

import (
	"context"

	"github.com/runletapp/crabfs/identity"

	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
)

// Relay controls a relay server
type Relay struct {
	ctx context.Context

	port uint

	host interfaces.Core
}

// RelayNew creates a new relay instance
func RelayNew(ctx context.Context, port uint, bootstrapPeers []string, id identity.Identity) (*Relay, error) {
	host, err := New(
		options.Port(port),
		options.RelayOnly(true),
		options.BootstrapPeers(bootstrapPeers),
		options.Identity(id),
	)
	if err != nil {
		return nil, err
	}

	relay := &Relay{
		ctx:  ctx,
		port: port,
		host: host,
	}

	return relay, nil
}

// Close closes this relay instance
func (relay *Relay) Close() error {
	return relay.host.Close()
}

// GetAddrs get the addresses that this node is bound to
func (relay *Relay) GetAddrs() []string {
	return relay.host.GetAddrs()
}

// GetHostID returns the id of this p2p relay
func (relay *Relay) GetHostID() string {
	return relay.host.GetID()
}

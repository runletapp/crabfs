package options

import (
	"context"

	"github.com/google/uuid"
	ipfsConfig "github.com/ipfs/go-ipfs-config"
)

// Settings init settings
type Settings struct {
	Context        context.Context
	Port           uint
	BootstrapPeers []string
	RelayOnly      bool

	Root string

	ID string
}

// Option represents a single init option
type Option func(s *Settings) error

// DefaultBootstrapPeers collection of peers to use as bootstrap by default
var DefaultBootstrapPeers = ipfsConfig.DefaultBootstrapAddresses

// SetDefaults set the default values
func (s *Settings) SetDefaults() error {
	s.Context = context.Background()
	s.Port = 0
	s.RelayOnly = false

	s.BootstrapPeers = DefaultBootstrapPeers

	s.Root = ""

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	s.ID = id.String()

	return nil
}

// Context option
func Context(ctx context.Context) Option {
	return func(s *Settings) error {
		s.Context = ctx
		return nil
	}
}

// Port option
func Port(port uint) Option {
	return func(s *Settings) error {
		s.Port = port
		return nil
	}
}

// BootstrapPeers option
func BootstrapPeers(peers []string) Option {
	return func(s *Settings) error {
		s.BootstrapPeers = peers
		return nil
	}
}

// BootstrapPeersAppend option
func BootstrapPeersAppend(peers []string) Option {
	return func(s *Settings) error {
		s.BootstrapPeers = append(s.BootstrapPeers, peers...)
		return nil
	}
}

// RelayOnly option
func RelayOnly(relayOnly bool) Option {
	return func(s *Settings) error {
		s.RelayOnly = relayOnly
		return nil
	}
}

// Root set base location
func Root(root string) Option {
	return func(s *Settings) error {
		s.Root = root
		return nil
	}
}

// ID sets the id of the peer
func ID(id string) Option {
	return func(s *Settings) error {
		s.ID = id
		return nil
	}
}

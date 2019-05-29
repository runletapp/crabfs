package options

import (
	"context"
	"io"
	"time"

	"github.com/runletapp/crabfs/identity"
)

// Settings init settings
type Settings struct {
	Context        context.Context
	Port           uint
	BootstrapPeers []string
	RelayOnly      bool

	BlockSize int64

	Root string

	ReprovideInterval time.Duration

	GCInterval time.Duration

	Identity identity.Identity
}

// Option represents a single init option
type Option func(s *Settings) error

// DefaultBootstrapPeers collection of peers to use as bootstrap by default
var DefaultBootstrapPeers = []string{}

// SetDefaults set the default values
func (s *Settings) SetDefaults() error {
	s.Context = context.Background()
	s.Port = 0
	s.RelayOnly = false

	s.BootstrapPeers = DefaultBootstrapPeers

	s.BlockSize = 100 * 1024

	s.Root = ""

	s.ReprovideInterval = 1 * time.Minute

	s.GCInterval = 1 * time.Hour

	// Defaults to create a new one
	s.Identity = nil

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

// BlockSize set the block site of this node
func BlockSize(blockSize int64) Option {
	return func(s *Settings) error {
		s.BlockSize = blockSize
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

// ReprovideInterval interval which blocks are republished to the network
func ReprovideInterval(interval time.Duration) Option {
	return func(s *Settings) error {
		s.ReprovideInterval = interval
		return nil
	}
}

// GCInterval interval which the garbage collector should run
func GCInterval(interval time.Duration) Option {
	return func(s *Settings) error {
		s.GCInterval = interval
		return nil
	}
}

// Identity set the node identity
func Identity(id identity.Identity) Option {
	return func(s *Settings) error {
		s.Identity = id
		return nil
	}
}

// IdentityFromReader set the node identity from a reader
func IdentityFromReader(r io.Reader) Option {
	return func(s *Settings) error {
		id, err := identity.UnmarshalIdentity(r)
		if err != nil {
			return err
		}

		s.Identity = id
		return nil
	}
}

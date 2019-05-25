package options

import (
	"bytes"
	"context"
	"io"
	"time"
)

// Settings init settings
type Settings struct {
	Context        context.Context
	Port           uint
	BootstrapPeers []string
	RelayOnly      bool

	PrivateKey io.Reader

	BlockSize int64

	Root string

	ReprovideInterval time.Duration

	GCInterval time.Duration
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
	s.PrivateKey = nil

	s.BootstrapPeers = DefaultBootstrapPeers

	s.BlockSize = 100 * 1024

	s.Root = ""

	s.ReprovideInterval = 1 * time.Minute

	s.GCInterval = 1 * time.Hour

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

// PrivateKey option
func PrivateKey(reader io.Reader) Option {
	return func(s *Settings) error {
		s.PrivateKey = reader
		return nil
	}
}

// PrivateKeyFromBytes option
func PrivateKeyFromBytes(privateKey []byte) Option {
	return func(s *Settings) error {
		r := bytes.NewReader(privateKey)
		s.PrivateKey = r
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

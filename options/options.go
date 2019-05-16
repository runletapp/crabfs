package options

import (
	"bytes"
	"context"
	"io"
)

// Settings init settings
type Settings struct {
	Context        context.Context
	BucketName     string
	Port           int
	BootstrapPeers []string
	RelayOnly      bool

	PrivateKey io.Reader

	BlockSize int64

	Root string
}

// Option represents a single init option
type Option func(s *Settings) error

// DefaultBootstrapPeers collection of peers to use as bootstrap by default
var DefaultBootstrapPeers = []string{}

// SetDefaults set the default values
func (s *Settings) SetDefaults() error {
	s.Context = context.Background()
	s.BucketName = "crabfs"
	s.Port = 0
	s.RelayOnly = false
	s.PrivateKey = nil

	s.BootstrapPeers = DefaultBootstrapPeers

	s.BlockSize = 100 * 1024

	s.Root = ""

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
func Port(port int) Option {
	return func(s *Settings) error {
		s.Port = port
		return nil
	}
}

// BucketName option
func BucketName(name string) Option {
	return func(s *Settings) error {
		s.BucketName = name
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

func Root(root string) Option {
	return func(s *Settings) error {
		s.Root = root
		return nil
	}
}

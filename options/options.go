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
	DiscoveryKey   string
	Port           int
	BootstrapPeers []string
	RelayOnly      bool

	SwarmKey io.Reader
}

// Option represents a single init option
type Option func(s *Settings) error

// DefaultBootstrapPeers collection of peers to use as bootstrap by default
var DefaultBootstrapPeers = []string{}

// SetDefaults set the default values
func (s *Settings) SetDefaults() error {
	s.Context = context.Background()
	s.BucketName = "crabfs"
	s.DiscoveryKey = "crabfs"
	s.Port = 0
	s.RelayOnly = false
	s.SwarmKey = nil

	// use ipfs default peers
	s.BootstrapPeers = DefaultBootstrapPeers

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

// DiscoveryKey option
func DiscoveryKey(key string) Option {
	return func(s *Settings) error {
		s.DiscoveryKey = key
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

// SwarmKey option
func SwarmKey(reader io.Reader) Option {
	return func(s *Settings) error {
		s.SwarmKey = reader
		return nil
	}
}

// SwarmKeyFromBytes option
func SwarmKeyFromBytes(privateKey []byte) Option {
	return func(s *Settings) error {
		r := bytes.NewReader(privateKey)
		s.SwarmKey = r
		return nil
	}
}

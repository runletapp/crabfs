package crabfs

import (
	"io"

	libp2pPnet "github.com/libp2p/go-libp2p-pnet"
)

// GenerateSwarmPrivateKey generates a private key ready to be used in the swarm protector
func GenerateSwarmPrivateKey() (io.Reader, error) {
	return libp2pPnet.GenerateV1PSK()
}

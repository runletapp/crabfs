package interfaces

import (
	"context"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"

	ipfsCoreApiIface "github.com/ipfs/interface-go-ipfs-core"
)

// Core base core interface
type Core interface {
	GetID() string
	GetAddrs() []string

	Close() error

	Create(ctx context.Context, privKey crabfsCrypto.PrivKey) (Bucket, error)
	Open(ctx context.Context, addr string, privKey crabfsCrypto.PrivKey) (Bucket, error)

	Storage() ipfsCoreApiIface.CoreAPI
}

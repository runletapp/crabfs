package identity

import (
	"crypto/rand"
	"io"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"

	pb "github.com/runletapp/crabfs/protos"

	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
)

var _ Identity = &Libp2pIdentity{}

// Libp2pIdentity identity provided by libp2p core
type Libp2pIdentity struct {
	pb.Identity

	key libp2pCrypto.PrivKey
}

// CreateIdentity creates a new identity
func CreateIdentity() (Identity, error) {
	key, _, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	if err != nil {
		return nil, err
	}

	data, err := libp2pCrypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, err
	}

	return &Libp2pIdentity{
		Identity: pb.Identity{
			PrivKey: data,
		},
		key: key,
	}, nil
}

// UnmarshalIdentity unmarshal identity from a reader
func UnmarshalIdentity(r io.Reader) (Identity, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var identity pb.Identity
	if err := proto.Unmarshal(data, &identity); err != nil {
		return nil, err
	}

	// Libp2pIdentity
	key, err := libp2pCrypto.UnmarshalPrivateKey(identity.PrivKey)
	if err != nil {
		return nil, err
	}

	libp2pID := Libp2pIdentity{
		Identity: identity,
		key:      key,
	}

	return &libp2pID, nil
}

func (id *Libp2pIdentity) Marshal() ([]byte, error) {
	return proto.Marshal(&id.Identity)
}

func (id *Libp2pIdentity) GetPrivateKey() []byte {
	return id.PrivKey
}

func (id *Libp2pIdentity) GetLibp2pPrivateKey() libp2pCrypto.PrivKey {
	return id.key
}

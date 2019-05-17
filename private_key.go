package crabfs

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"

	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
)

// GenerateKeyPair generates a private and public keys ready to be used in the swarm
func GenerateKeyPair() (io.Reader, error) {
	privateKey, _, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	if err != nil {
		return nil, err
	}

	data, err := libp2pCrypto.MarshalPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(data)
	return reader, nil
}

// ReadPrivateKey reads the key from a reader
func ReadPrivateKey(in io.Reader) (*libp2pCrypto.RsaPrivateKey, error) {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	key, err := libp2pCrypto.UnmarshalRsaPrivateKey(data)
	if err != nil {
		return nil, err
	}

	return key.(*libp2pCrypto.RsaPrivateKey), nil
}

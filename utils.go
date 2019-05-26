package crabfs

import (
	"fmt"

	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	multihash "github.com/multiformats/go-multihash"
)

// KeyFromFilename converts a file name to a key
func KeyFromFilename(pk string, bucketFilename string) string {
	hash, _ := multihash.Sum([]byte(bucketFilename), multihash.SHA3_256, -1)
	return fmt.Sprintf("/crabfs/v1/%s/%s", pk, hash.String())
}

// PublicKeyHashFromPrivateKey returns the hash of the private key
func PublicKeyHashFromPrivateKey(privateKey libp2pCrypto.PrivKey) (string, error) {
	publicKey := privateKey.GetPublic()
	return PublicKeyHashFromPublicKey(publicKey)
}

// PublicKeyHashFromPublicKey returns the hash of the public key
func PublicKeyHashFromPublicKey(publicKey libp2pCrypto.PubKey) (string, error) {
	publicKeyData, err := libp2pCrypto.MarshalPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	publicKeyHash, err := multihash.Sum(publicKeyData, multihash.SHA3_256, -1)
	if err != nil {
		return "", err
	}

	return publicKeyHash.String(), nil
}

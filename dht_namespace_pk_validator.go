package crabfs

import (
	"fmt"
	"strings"

	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	libp2pRecord "github.com/libp2p/go-libp2p-record"
	multihash "github.com/multiformats/go-multihash"
)

// DHTNamespacePKValidatorNew creates a new validator that validates for all versions
func DHTNamespacePKValidatorNew() libp2pRecord.Validator {
	return DHTNamespacePKValidatorV1{}
}

// DHTNamespacePKValidatorV1 validates the /crabfs keys on the dht datastore
type DHTNamespacePKValidatorV1 struct {
}

// Validate validates the given record, returning an error if it's
// invalid (e.g., expired, signed by the wrong key, etc.).
func (validator DHTNamespacePKValidatorV1) Validate(key string, value []byte) error {
	parts := strings.Split(key, "/")

	if len(parts) != 3 {
		return fmt.Errorf("Invalid key. Expexted format: /crabfs_pk/<hash> Got: %v", key)
	}

	if _, err := libp2pCrypto.UnmarshalPublicKey(value); err != nil {
		return err
	}

	pskHash, err := multihash.Sum(value, multihash.SHA3_256, -1)
	if err != nil {
		return err
	}

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", pskHash.String())
	if !strings.HasPrefix(key, expectedKey) {
		return fmt.Errorf("Invalid key. Expexted: %s Got: %v", expectedKey, key)
	}

	return nil
}

// Select selects the best record from the set of records (e.g., the
// newest).
//
// Decisions made by select should be stable.
func (validator DHTNamespacePKValidatorV1) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("No values")
	}

	return 0, nil
}

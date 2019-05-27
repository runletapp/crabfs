package crabfs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	libp2pRecord "github.com/libp2p/go-libp2p-record"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
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

	if _, err := crabfsCrypto.UnmarshalPublicKey(value); err != nil {
		return err
	}

	hash := sha256.New()
	_, err := hash.Write(value)
	if err != nil {
		return err
	}

	pskHash := hash.Sum(nil)

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", hex.EncodeToString(pskHash))
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

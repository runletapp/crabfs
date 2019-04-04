package crabfs

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	libp2pRecord "github.com/libp2p/go-libp2p-record"
)

// DHTNamespaceValidatorNew creates a new validator that validates for all versions
func DHTNamespaceValidatorNew(discoveryKey string) libp2pRecord.Validator {
	return DHTNamespaceValidatorV1{
		discoveryKey: discoveryKey,
	}
}

// DHTNamespaceValidatorV1 validates the /crabfs keys on the dht datastore
type DHTNamespaceValidatorV1 struct {
	discoveryKey string
}

// Validate validates the given record, returning an error if it's
// invalid (e.g., expired, signed by the wrong key, etc.).
func (validator DHTNamespaceValidatorV1) Validate(key string, value []byte) error {
	if !strings.HasPrefix(key, fmt.Sprintf("/crabfs/%s/", validator.discoveryKey)) {
		return fmt.Errorf("Invalid key")
	}

	parts := strings.Split(key, "/")

	if len(parts) != 4 {
		return fmt.Errorf("Invalid key")
	}

	var record DHTNameRecord
	if err := proto.Unmarshal(value, &record); err != nil {
		return err
	}

	if _, err := time.Parse(time.RFC3339, record.Timestamp); err != nil {
		return err
	}

	_, err := cid.Cast(record.ContentID)
	if err != nil {
		return err
	}

	return nil
}

// Select selects the best record from the set of records (e.g., the
// newest).
//
// Decisions made by select should be stable.
func (validator DHTNamespaceValidatorV1) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("No values")
	}

	var selected int
	selected = -1

	var value []byte
	var record DHTNameRecord
	var lastRecord *DHTNameRecord
	for i := 0; i < len(values); i++ {
		if err := proto.Unmarshal(value, &record); err != nil {
			return 0, err
		}

		if lastRecord == nil {
			selected = i
		} else if bytes.Compare([]byte(lastRecord.Timestamp), []byte(record.Timestamp)) > 0 {
			selected = i
		}
		lastRecord = &record
	}

	return selected, nil
}

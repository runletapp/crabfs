package crabfs

import cid "github.com/ipfs/go-cid"

// DHTNameRecord dht record to store on nodes
type DHTNameRecord struct {
	DHTNameRecordV1

	ContentID *cid.Cid
}

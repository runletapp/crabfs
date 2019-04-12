package crabfs

import (
	multihash "github.com/multiformats/go-multihash"

	cid "github.com/ipfs/go-cid"
)

// Object common object to be stored in the fs
type Object interface {
	// CalcCID return the CID of this object
	CalcCID() (*cid.Cid, error)

	// CalcHash return the current hash of the object
	CalcHash() (multihash.Multihash, error)

	// GetType returns the object type
	GetType() ObjectType

	// GetSize returns the size of the object
	GetSize() int64
}

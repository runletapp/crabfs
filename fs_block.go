package crabfs

import (
	"io"
	"time"

	multihash "github.com/multiformats/go-multihash"

	cid "github.com/ipfs/go-cid"
)

// BlockType Defines the block type
type BlockType int

var (
	// BlockTypeFILE file type
	BlockTypeFILE BlockType = 0x01
)

// Block common object to be stored in the fs
type Block struct {
	name string

	mhash multihash.Multihash

	Perm int

	Type BlockType
}

// GetCID return the CID of this block
func (block *Block) GetCID() cid.Cid {
	return cid.NewCidV1(cid.Raw, block.mhash)
}

// CalcHash return the current hash of the block
func (block *Block) CalcHash(mtime *time.Time) (multihash.Multihash, error) {
	return nil, io.EOF
}

package crabfs

import (
	"bytes"
	"os"

	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	"gopkg.in/src-d/go-billy.v4"
)

// Dir p2p directory abstraction
type Dir struct {
	Block

	billy.Dir

	Children []*cid.Cid
}

// DirNew creates a new directory wrapper
func DirNew(dirname string) (*Dir, error) {
	return &Dir{
		Block: Block{
			name: dirname,
			Type: BlockTypeDIR,
		},

		Children: []*cid.Cid{},
	}, nil
}

// CalcHash return the current hash of the block
func (dir *Dir) CalcHash() (multihash.Multihash, error) {
	var buffer bytes.Buffer

	// dir hash is composed of the dirname + children cid's
	_, err := buffer.Write([]byte(dir.name))
	if err != nil {
		return nil, err
	}

	for _, cid := range dir.Children {
		_, err := buffer.Write(cid.Bytes())
		if err != nil {
			return nil, err
		}
	}

	hash, err := multihash.Sum(buffer.Bytes(), multihash.SHA3_256, -1)
	if err != nil {
		return nil, err
	}
	dir.mhash = hash

	return hash, nil
}

// Name returns the name of the dir as presented to Open.
func (dir *Dir) Name() string {
	return dir.name
}

// ReadDir reads the directory named by dirname and returns a list of
// directory entries sorted by filename.
func (dir *Dir) ReadDir(path string) ([]os.FileInfo, error) {
	return dir.Dir.ReadDir(path)
}

// MkdirAll creates a directory named path, along with any necessary
// parents, and returns nil, or else returns an error. The permission bits
// perm are used for all directories that MkdirAll creates. If path is/
// already a directory, MkdirAll does nothing and returns nil.
func (dir *Dir) MkdirAll(filename string, perm os.FileMode) error {
	return dir.Dir.MkdirAll(filename, perm)
}

package interfaces

import (
	pb "github.com/runletapp/crabfs/protos"

	blocks "github.com/ipfs/go-block-format"
)

// Slicer slice a reader into a series of blocks
type Slicer interface {
	// Next return the next block or nil if it has reached eof
	Next() (*pb.BlockMetadata, blocks.Block, int64, error)
}

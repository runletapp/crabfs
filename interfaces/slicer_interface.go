package interfaces

import (
	pb "github.com/runletapp/crabfs/protos"
)

// Slicer slice a reader into a series of blocks
type Slicer interface {
	// Next return the next block or nil if it has reached eof
	Next() (*pb.BlockMetadata, []byte, int64, error)
}

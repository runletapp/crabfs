package crabfs

import (
	"io"

	blocks "github.com/ipfs/go-block-format"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"
)

var _ interfaces.Slicer = &BlockSlicer{}

// BlockSlicer a simple block slicer
type BlockSlicer struct {
	reader io.Reader

	blockSize int64

	offset int64
}

// BlockSlicerNew creates a new BlockSlicer with 'blockSize'
func BlockSlicerNew(reader io.Reader, blockSize int64) (interfaces.Slicer, error) {
	slicer := &BlockSlicer{
		reader:    reader,
		blockSize: blockSize,

		offset: 0,
	}

	return slicer, nil
}

func (slicer *BlockSlicer) Next() (*pb.BlockMetadata, blocks.Block, error) {
	data := make([]byte, slicer.blockSize)

	n, err := slicer.reader.Read(data)
	if err != nil {
		return nil, nil, err
	}

	block := blocks.NewBlock(data[:n])

	metadata := &pb.BlockMetadata{
		Start: slicer.offset,
		Size:  int64(n),
		Cid:   block.Cid().Bytes(),
	}

	slicer.offset += int64(n)

	return metadata, block, nil
}

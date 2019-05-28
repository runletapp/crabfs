package crabfs

import (
	"io"

	"crypto/cipher"

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

	cipher cipher.Block
}

// BlockSlicerNew creates a new BlockSlicer with 'blockSize'
func BlockSlicerNew(reader io.Reader, blockSize int64, cipher cipher.Block) (interfaces.Slicer, error) {
	slicer := &BlockSlicer{
		reader:    reader,
		blockSize: blockSize,

		offset: 0,

		cipher: cipher,
	}

	return slicer, nil
}

func (slicer *BlockSlicer) Next() (*pb.BlockMetadata, blocks.Block, int64, error) {
	data := make([]byte, slicer.blockSize)

	n, err := slicer.reader.Read(data)
	if err != nil {
		return nil, nil, 0, err
	}

	dataBlock := append([]byte{}, data[:n]...)

	paddingStart := int64(n)

	for i := 0; i < slicer.cipher.BlockSize()-n; i++ {
		dataBlock = append(dataBlock, 0)
	}

	slicer.cipher.Encrypt(dataBlock, dataBlock)

	block := blocks.NewBlock(dataBlock)

	metadata := &pb.BlockMetadata{
		Start:        slicer.offset,
		Size:         int64(len(dataBlock)),
		Cid:          block.Cid().Bytes(),
		PaddingStart: paddingStart,
	}

	slicer.offset += int64(n)

	return metadata, block, int64(n), nil
}

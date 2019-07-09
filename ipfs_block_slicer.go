package crabfs

import (
	"io"

	"crypto/cipher"

	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"

	ipfsChunker "github.com/ipfs/go-ipfs-chunker"
)

var _ interfaces.Slicer = &IPFSBlockSlicer{}

// IPFSBlockSlicer a simple block slicer
type IPFSBlockSlicer struct {
	chunker ipfsChunker.Splitter

	offset int64

	cipher cipher.Block
}

// IPFSBlockSlicerNew creates a new BlockSlicer using the default ipfs splitter
func IPFSBlockSlicerNew(reader io.Reader, cipher cipher.Block) (interfaces.Slicer, error) {
	slicer := &IPFSBlockSlicer{
		chunker: ipfsChunker.DefaultSplitter(reader),

		offset: 0,

		cipher: cipher,
	}

	return slicer, nil
}

func (slicer *IPFSBlockSlicer) Next() (*pb.BlockMetadata, []byte, int64, error) {
	data, err := slicer.chunker.NextBytes()
	if err != nil {
		return nil, nil, 0, err
	}

	n := len(data)

	paddingStart := int64(n)

	for i := 0; i < slicer.cipher.BlockSize()-n; i++ {
		data = append(data, 0)
	}

	slicer.cipher.Encrypt(data, data)

	metadata := &pb.BlockMetadata{
		Start:        slicer.offset,
		Size:         int64(len(data)),
		PaddingStart: paddingStart,
	}

	slicer.offset += int64(n)

	return metadata, data, int64(n), nil
}

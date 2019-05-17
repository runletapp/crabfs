package crabfs

import (
	"bytes"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/stretchr/testify/assert"
)

func setUpBlockSlicerTest(t *testing.T, reader io.Reader, blockSize int64) (interfaces.Slicer, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	slicer, err := BlockSlicerNew(reader, blockSize)
	assert.Nil(err)

	return slicer, ctrl
}

func setDownBlockSlicerTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestBlockSliceSmallBlocks(t *testing.T) {
	assert := assert.New(t)

	buffer := &bytes.Buffer{}
	buffer.Write([]byte("abc"))

	slicer, ctrl := setUpBlockSlicerTest(t, buffer, 500*1024)
	defer setDownBlockSlicerTest(ctrl)

	meta, block, err := slicer.Next()
	assert.Nil(err)

	assert.NotNil(meta)
	assert.NotNil(block)

	assert.Equal(int64(3), meta.Size)
	assert.Equal(int64(0), meta.Start)

	expectedBlock := blocks.NewBlock([]byte("abc"))
	assert.True(expectedBlock.Cid().Equals(block.Cid()))

	meta, block, err = slicer.Next()
	assert.Nil(meta)
	assert.Nil(block)

	assert.NotNil(err)
}

func TestBlockSliceBigBlocks(t *testing.T) {
	assert := assert.New(t)

	buffer := &bytes.Buffer{}
	buffer.Write([]byte("0123456789"))

	slicer, ctrl := setUpBlockSlicerTest(t, buffer, 2)
	defer setDownBlockSlicerTest(ctrl)

	meta, block, err := slicer.Next()
	assert.Nil(err)

	assert.NotNil(meta)
	assert.NotNil(block)

	assert.Equal(int64(2), meta.Size)
	assert.Equal(int64(0), meta.Start)

	for count := 1; count < 5; count++ {
		meta, block, err = slicer.Next()
		assert.Nil(err)

		assert.NotNil(meta)
		assert.NotNil(block)

		assert.Equal(int64(2), meta.Size)
		assert.Equal(int64(count*2), meta.Start)
	}

	meta, block, err = slicer.Next()
	assert.Nil(meta)
	assert.Nil(block)

	assert.NotNil(err)
}

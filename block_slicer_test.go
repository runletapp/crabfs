package crabfs

import (
	"bytes"
	"crypto/aes"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/stretchr/testify/assert"
)

func setUpBlockSlicerTest(t *testing.T, reader io.Reader, blockSize int64) (interfaces.Slicer, *gomock.Controller) {
	slicer, _, ctrl := setUpBlockSlicerTestWithPrivateKey(t, reader, blockSize)
	return slicer, ctrl
}

func setUpBlockSlicerTestWithPrivateKey(t *testing.T, reader io.Reader, blockSize int64) (interfaces.Slicer, crabfsCrypto.PrivKey, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	privateKey, err := GenerateKeyPair()
	assert.Nil(err)

	cipher, err := aes.NewCipher([]byte("0123456789abcdef"))
	assert.Nil(err)

	slicer, err := BlockSlicerNew(reader, blockSize, cipher)
	assert.Nil(err)

	return slicer, privateKey, ctrl
}

func setDownBlockSlicerTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestBlockSliceSmallBlocks(t *testing.T) {
	assert := assert.New(t)

	buffer := &bytes.Buffer{}
	buffer.Write([]byte("abc"))

	slicer, _, ctrl := setUpBlockSlicerTestWithPrivateKey(t, buffer, 500*1024)
	defer setDownBlockSlicerTest(ctrl)

	meta, block, n, err := slicer.Next()
	assert.Nil(err)

	assert.NotNil(meta)
	assert.NotNil(block)

	assert.Equal(int64(3), n)
	assert.Equal(int64(0), meta.Start)

	assert.NotEqual([]byte("abc"), block.RawData())

	meta, block, n, err = slicer.Next()
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

	meta, block, n, err := slicer.Next()
	assert.Nil(err)

	assert.NotNil(meta)
	assert.NotNil(block)

	assert.True(meta.Size > 0)
	assert.Equal(int64(2), n)
	assert.Equal(int64(0), meta.Start)

	for count := 1; count < 5; count++ {
		meta, block, n, err = slicer.Next()
		assert.Nil(err)

		assert.NotNil(meta)
		assert.NotNil(block)

		assert.True(meta.Size > 0)
		assert.Equal(int64(2), n)
		assert.Equal(int64(count*2), meta.Start)
	}

	meta, block, n, err = slicer.Next()
	assert.Nil(meta)
	assert.Nil(block)
	assert.Equal(int64(0), n)

	assert.NotNil(err)
}

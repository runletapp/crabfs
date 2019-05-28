package crabfs

import (
	"crypto/aes"
	"testing"

	"github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
)

func setUpBasicTest(t *testing.T) *gomock.Controller {
	ctrl := gomock.NewController(t)

	return setUpBasicTestWithCtrl(t, ctrl)
}

func setUpBasicTestWithCtrl(t *testing.T, ctrl *gomock.Controller) *gomock.Controller {
	return ctrl
}

func setDownBasicTest(ctrl *gomock.Controller) {
	ctrl.Finish()
}

func CreateBlockFromString(data string, key []byte) (blocks.Block, int64) {
	cipher, _ := aes.NewCipher(key)

	dataEnc := make([]byte, len(data))
	copy(dataEnc, []byte(data))

	paddingStart := len(dataEnc)
	for i := 0; i < cipher.BlockSize()-paddingStart; i++ {
		dataEnc = append(dataEnc, 0)
	}

	cipher.Encrypt(dataEnc, dataEnc)

	return blocks.NewBlock(dataEnc), int64(paddingStart)
}

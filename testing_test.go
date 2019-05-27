package crabfs

import (
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

func CreateBlockFromString(data string) blocks.Block {
	return blocks.NewBlock([]byte(data))
}

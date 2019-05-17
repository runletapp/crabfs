package crabfs

import (
	"crypto/rand"
	"testing"

	"github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	multihash "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
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

func GenerateKeyPairWithHash(t *testing.T) (*libp2pCrypto.RsaPrivateKey, *libp2pCrypto.RsaPublicKey, multihash.Multihash) {
	assert := assert.New(t)

	privKey, publicKey, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	assert.Nil(err)

	data, err := libp2pCrypto.MarshalRsaPublicKey(publicKey.(*libp2pCrypto.RsaPublicKey))
	assert.Nil(err)

	pskHash, err := multihash.Sum(data, multihash.SHA3_256, -1)
	assert.Nil(err)

	return privKey.(*libp2pCrypto.RsaPrivateKey), publicKey.(*libp2pCrypto.RsaPublicKey), pskHash
}

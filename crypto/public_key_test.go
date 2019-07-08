package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setUpPubKeyTest(t *testing.T) (PrivKey, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	assert := assert.New(t)

	key, err := PrivateKeyNew(rand.Reader, 1024)
	assert.Nil(err)

	return key, ctrl
}

func setDownPubKeyTest(ctrl *gomock.Controller) {
	ctrl.Finish()
}

func TestPubKeyMarshal(t *testing.T) {
	key, ctrl := setUpPubKeyTest(t)
	defer setDownPubKeyTest(ctrl)
	assert := assert.New(t)

	data, err := key.GetPublic().Marshal()
	assert.Nil(err)
	assert.True(len(data) > 0)
}

func TestPubKeyUnMarshal(t *testing.T) {
	key, ctrl := setUpPubKeyTest(t)
	defer setDownPubKeyTest(ctrl)
	assert := assert.New(t)

	r, err := key.GetPublic().Marshal()
	assert.Nil(err)

	key2, err := UnmarshalPublicKey(r)
	assert.Nil(err)

	assert.True(bytes.Compare(key.GetPublic().Hash(), key2.Hash()) == 0)
}

func TestPubKeyUnMarshalInvalid(t *testing.T) {
	_, ctrl := setUpPubKeyTest(t)
	defer setDownPubKeyTest(ctrl)
	assert := assert.New(t)

	_, err := UnmarshalPublicKey([]byte("abc"))
	assert.NotNil(err)
}

package crypto

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setUpPrivKeyTest(t *testing.T) (PrivKey, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	assert := assert.New(t)

	key, err := PrivateKeyNew(rand.Reader, 1024)
	assert.Nil(err)

	return key, ctrl
}

func setDownPrivKeyTest(ctrl *gomock.Controller) {
	ctrl.Finish()
}

func TestPrivKeyMarshall(t *testing.T) {
	key, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	r, err := key.Marshall()
	assert.Nil(err)

	data, err := ioutil.ReadAll(r)
	assert.Nil(err)
	assert.True(len(data) > 0)
}

func TestPrivKeyUnMarshall(t *testing.T) {
	key, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	r, err := key.Marshall()
	assert.Nil(err)

	key2, err := UnmarshallPrivateKey(r)
	assert.Nil(err)

	assert.True(bytes.Compare(key.Hash(), key2.Hash()) == 0)
}

func TestPrivKeyUnMarshallInvalid(t *testing.T) {
	_, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	_, err := UnmarshallPrivateKey(bytes.NewReader([]byte("abc")))
	assert.NotNil(err)
}

func TestPrivKeyEncryptDecrypt(t *testing.T) {
	key, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	cipher, err := key.GetPublic().Encrypt([]byte("abc"), []byte("label"))
	assert.Nil(err)

	assert.False(bytes.Compare(cipher, []byte("abc")) == 0)

	data, err := key.Decrypt(cipher, []byte("label"))
	assert.Nil(err)

	assert.Equal(data, []byte("abc"))
}

func TestPrivKeyEncryptDecryptWrongLabel(t *testing.T) {
	key, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	cipher, err := key.GetPublic().Encrypt([]byte("abc"), []byte("label"))
	assert.Nil(err)

	assert.False(bytes.Compare(cipher, []byte("abc")) == 0)

	_, err = key.Decrypt(cipher, []byte("label2"))
	assert.NotNil(err)
}

func TestPrivKeySignValidate(t *testing.T) {
	key, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	sign, err := key.Sign([]byte("abc"))
	assert.Nil(err)

	check, err := key.GetPublic().Validate([]byte("abc"), sign)
	assert.Nil(err)
	assert.True(check)
}

func TestPrivKeySignValidateInvalid(t *testing.T) {
	key, ctrl := setUpPrivKeyTest(t)
	defer setDownPrivKeyTest(ctrl)
	assert := assert.New(t)

	sign, err := key.Sign([]byte("abc"))
	assert.Nil(err)

	check, err := key.GetPublic().Validate([]byte("abc2"), sign)
	assert.NotNil(err)
	assert.False(check)
}

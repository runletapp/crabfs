package crabfs

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	libp2pRecord "github.com/libp2p/go-libp2p-record"
	multihash "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
)

func setUpPkValidatorTest(t *testing.T) (libp2pRecord.Validator, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	validator := DHTNamespacePKValidatorNew()

	return validator, ctrl
}

func setDownPkValidatorTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestPkValidatorValidateInvalidKey(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	assert.NotNil(validator.Validate("", nil))
}

func TestPkValidatorValidateInvalidPublicKey(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	assert.NotNil(validator.Validate("/crabfs_pk/12345", nil))
	assert.NotNil(validator.Validate("/crabfs_pk/12345", []byte("abc")))
}

func TestPkValidatorValidateInvalidPublicKeyHash(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	_, publicKey, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	assert.Nil(err)

	data, err := libp2pCrypto.MarshalRsaPublicKey(publicKey.(*libp2pCrypto.RsaPublicKey))
	assert.Nil(err)

	assert.NotNil(validator.Validate("/crabfs_pk/12345", data))
}

func TestPkValidatorValidateValidPublicKey(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	_, publicKey, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	assert.Nil(err)

	data, err := libp2pCrypto.MarshalRsaPublicKey(publicKey.(*libp2pCrypto.RsaPublicKey))
	assert.Nil(err)

	pskHash, err := multihash.Sum(data, multihash.SHA3_256, -1)
	assert.Nil(err)

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", pskHash.String())

	assert.Nil(validator.Validate(expectedKey, data))
}

func TestPkValidatorSelect(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	_, publicKey, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	assert.Nil(err)

	data, err := libp2pCrypto.MarshalRsaPublicKey(publicKey.(*libp2pCrypto.RsaPublicKey))
	assert.Nil(err)

	pskHash, err := multihash.Sum(data, multihash.SHA3_256, -1)
	assert.Nil(err)

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", pskHash.String())

	i, err := validator.Select(expectedKey, [][]byte{
		data,
		data,
	})

	assert.Nil(err)
	assert.Equal(0, i)
}

func TestPkValidatorSelectNoValues(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	_, publicKey, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
	assert.Nil(err)

	data, err := libp2pCrypto.MarshalRsaPublicKey(publicKey.(*libp2pCrypto.RsaPublicKey))
	assert.Nil(err)

	pskHash, err := multihash.Sum(data, multihash.SHA3_256, -1)
	assert.Nil(err)

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", pskHash.String())

	_, err = validator.Select(expectedKey, [][]byte{})

	assert.NotNil(err)
}

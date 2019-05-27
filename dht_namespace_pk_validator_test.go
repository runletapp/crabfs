package crabfs

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	libp2pRecord "github.com/libp2p/go-libp2p-record"
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

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	publicKey := privKey.GetPublic()

	data, err := publicKey.Marshal()
	assert.Nil(err)

	assert.NotNil(validator.Validate("/crabfs_pk/12345", data))
}

func TestPkValidatorValidateValidPublicKey(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	publicKey := privKey.GetPublic()

	data, err := publicKey.Marshal()
	assert.Nil(err)

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", publicKey.HashString())

	assert.Nil(validator.Validate(expectedKey, data))
}

func TestPkValidatorSelect(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpPkValidatorTest(t)
	defer setDownPkValidatorTest(ctrl)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	publicKey := privKey.GetPublic()

	data, err := publicKey.Marshal()
	assert.Nil(err)

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", publicKey.HashString())

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

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	publicKey := privKey.GetPublic()

	expectedKey := fmt.Sprintf("/crabfs_pk/%s", publicKey.HashString())

	_, err = validator.Select(expectedKey, [][]byte{})

	assert.NotNil(err)
}

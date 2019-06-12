package crabfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)

	fs, err := New()
	assert.Nil(err)

	assert.NotEmpty(fs.GetID())
	assert.True(len(fs.GetAddrs()) > 0)

	assert.NotNil(fs.Blockstore())
	assert.NotNil(fs.Host())
	assert.NotNil(fs.GarbageCollector())
	assert.NotNil(fs.GetIdentity())

	assert.Nil(fs.Close())
}

func TestBucket(t *testing.T) {
	assert := assert.New(t)

	fs, err := New()
	assert.Nil(err)
	defer fs.Close()

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	bucket, err := fs.WithBucket(privKey, "bkt")
	assert.Nil(err)
	assert.NotNil(bucket)
}

func TestBucketChroot(t *testing.T) {
	assert := assert.New(t)

	fs, err := New()
	assert.Nil(err)
	defer fs.Close()

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	bucket, err := fs.WithBucketRoot(privKey, "bkt", "/data")
	assert.Nil(err)
	assert.NotNil(bucket)
}

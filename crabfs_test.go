package crabfs

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/runletapp/crabfs/options"
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

func TestPut(t *testing.T) {
	assertParent := assert.New(t)

	privKey, err := GenerateKeyPair()
	assertParent.Nil(err)

	relay, err := RelayNew(context.Background(), 0, []string{}, nil)
	assertParent.Nil(err)

	t.Run("NoPeers", func(t *testing.T) {
		assert := assert.New(t)

		fs, err := New()
		assert.Nil(err)
		defer fs.Close()

		reader := bytes.NewReader([]byte("data"))
		assert.Nil(fs.Put(context.Background(), privKey, "nopeers", "test.txt", reader, time.Now()))

		fetcher, err := fs.Get(context.Background(), privKey, "nopeers", "test.txt")
		assert.Nil(err)
		defer fetcher.Close()

		data, err := ioutil.ReadAll(fetcher)
		assert.Nil(err)
		assert.Equal([]byte("data"), data)
	})

	t.Run("relay p2p", func(t *testing.T) {
		assert := assert.New(t)

		fs1, err := New(options.BootstrapPeers(relay.GetAddrs()))
		assert.Nil(err)
		defer fs1.Close()

		assert.Nil(fs1.PublishPublicKey(privKey.GetPublic()))

		fs2, err := New(options.BootstrapPeers(relay.GetAddrs()))
		assert.Nil(err)
		defer fs2.Close()

		assert.Nil(fs2.PublishPublicKey(privKey.GetPublic()))

		reader := bytes.NewReader([]byte("data"))
		assert.Nil(fs1.Put(context.Background(), privKey, "relay p2p", "test.txt", reader, time.Now()))

		fetcher, err := fs2.Get(context.Background(), privKey, "relay p2p", "test.txt")
		assert.Nil(err)
		if err != nil {
			t.FailNow()
		}
		defer fetcher.Close()

		data, err := ioutil.ReadAll(fetcher)
		assert.Nil(err)
		assert.Equal([]byte("data"), data)
	})
}

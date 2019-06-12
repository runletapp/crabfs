package crabfs

import (
	"context"
	"testing"
	"time"

	blocks "github.com/ipfs/go-block-format"
	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/runletapp/crabfs/identity"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"
	"github.com/stretchr/testify/assert"
)

func setUpHostBenchWithRelay(t *testing.B) (interfaces.Host, ipfsBlockstore.Blockstore) {
	assert := assert.New(t)
	id, err := identity.CreateIdentity()
	assert.Nil(err)

	settings := &options.Settings{
		Context:  context.Background(),
		Identity: id,
	}

	ds := ipfsDatastore.NewMapDatastore()
	bs := ipfsBlockstore.NewBlockstore(ds)

	relay, err := RelayNew(settings.Context, 0, []string{}, nil)
	assert.Nil(err)

	settings.BootstrapPeers = relay.GetAddrs()

	host, err := HostNew(settings, ds, bs)
	assert.Nil(err)

	t.ResetTimer()

	return host, bs
}

func setUpHostBench(t *testing.B) (interfaces.Host, ipfsBlockstore.Blockstore) {
	assert := assert.New(t)
	id, err := identity.CreateIdentity()
	assert.Nil(err)

	settings := &options.Settings{
		Context:  context.Background(),
		Identity: id,
	}

	ds := ipfsDatastore.NewMapDatastore()
	bs := ipfsBlockstore.NewBlockstore(ds)

	host, err := HostNew(settings, ds, bs)
	assert.Nil(err)

	t.ResetTimer()

	return host, bs
}

func BenchmarkHostPublishPeers(t *testing.B) {
	assert := assert.New(t)
	host, bs := setUpHostBenchWithRelay(t)

	assert.Nil(host.Announce())

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		block := blocks.NewBlock([]byte("abc"))
		bs.Put(block)

		blockMap := interfaces.BlockMap{}
		blockMap[0] = &pb.BlockMetadata{
			Start: 0,
			Size:  int64(len(block.RawData())),
			Cid:   block.Cid().Bytes(),
		}

		cipherKey := []byte("0123456789abcdef")

		assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))
	}
}

func BenchmarkHostPublishNoPeers(t *testing.B) {
	assert := assert.New(t)
	host, bs := setUpHostBench(t)

	assert.Nil(host.Announce())

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		block := blocks.NewBlock([]byte("abc"))
		bs.Put(block)

		blockMap := interfaces.BlockMap{}
		blockMap[0] = &pb.BlockMetadata{
			Start: 0,
			Size:  int64(len(block.RawData())),
			Cid:   block.Cid().Bytes(),
		}

		cipherKey := []byte("0123456789abcdef")

		assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))
	}
}

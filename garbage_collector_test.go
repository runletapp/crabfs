package crabfs

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreSync "github.com/ipfs/go-datastore/sync"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"
	"github.com/stretchr/testify/assert"
)

func setUpGarbageCollectorTest(ctx context.Context, t *testing.T, interval time.Duration) (interfaces.GarbageCollector, ipfsDatastore.Batching, ipfsBlockstore.Blockstore, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	ds := ipfsDatastoreSync.MutexWrap(ipfsDatastore.NewMapDatastore())

	bs := ipfsBlockstore.NewBlockstore(ds)

	gc, err := GarbageCollectorNew(ctx, interval, ds, bs)
	assert.Nil(err)

	return gc, ds, bs, ctrl
}

func setDownGarbageCollectorTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestGarbageCollectorStopWithCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, _, _, ctrl := setUpGarbageCollectorTest(ctx, t, 1*time.Hour)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	assert.Nil(gc.Start())
}

func TestGarbageCollectorSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, _, _, ctrl := setUpGarbageCollectorTest(ctx, t, 1*time.Hour)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	assert.Nil(gc.Start())

	assert.Nil(gc.Schedule())
	assert.Nil(gc.Schedule())
}

func TestGarbageCollectorTicker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, _, _, ctrl := setUpGarbageCollectorTest(ctx, t, 500*time.Millisecond)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	assert.Nil(gc.Start())

	<-time.After(1 * time.Second)
}

func TestGarbageCollectorCollectEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, _, _, ctrl := setUpGarbageCollectorTest(ctx, t, 1*time.Hour)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	assert.Nil(gc.Collect())
}

func TestGarbageCollectorCollectWithBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, _, bs, ctrl := setUpGarbageCollectorTest(ctx, t, 1*time.Hour)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	blocks := []blocks.Block{
		blocks.NewBlock([]byte("abc")),
		blocks.NewBlock([]byte("abc2")),
	}
	assert.Nil(bs.PutMany(blocks))

	prs, err := bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	assert.Nil(gc.Collect())

	prs, err = bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(false, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(false, prs)
}

func TestGarbageCollectorCollectWithUsedBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, ds, bs, ctrl := setUpGarbageCollectorTest(ctx, t, 1*time.Hour)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	blocks := []blocks.Block{
		blocks.NewBlock([]byte("abc")),
		blocks.NewBlock([]byte("abc2")),
		blocks.NewBlock([]byte("abc3")),
	}
	assert.Nil(bs.PutMany(blocks))

	recordValue := pb.DHTNameRecordValue{
		Blocks: map[int64]*pb.BlockMetadata{
			0: &pb.BlockMetadata{
				Cid:   blocks[0].Cid().Bytes(),
				Start: 0,
				Size:  int64(len(blocks[0].RawData())),
			},
			1: &pb.BlockMetadata{
				Cid:   blocks[1].Cid().Bytes(),
				Start: 3,
				Size:  int64(len(blocks[0].RawData())),
			},
		},
	}
	recordValueData, err := proto.Marshal(&recordValue)
	assert.Nil(err)

	record := pb.DHTNameRecord{
		Data: recordValueData,
	}
	recordData, err := proto.Marshal(&record)
	assert.Nil(err)

	// Blocks 0 and 1 are now used
	key := ipfsDatastore.NewKey("/crabfs/abc")
	assert.Nil(ds.Put(key, recordData))

	prs, err := bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[2].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	// Collect
	assert.Nil(gc.Collect())

	prs, err = bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[2].Cid())
	assert.Nil(err)
	assert.Equal(false, prs)
}

func TestGarbageCollectorCollectLocker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc, _, bs, ctrl := setUpGarbageCollectorTest(ctx, t, 1*time.Hour)
	defer setDownGarbageCollectorTest(ctrl)
	assert := assert.New(t)

	blocks := []blocks.Block{
		blocks.NewBlock([]byte("abc")),
		blocks.NewBlock([]byte("abc2")),
	}
	assert.Nil(bs.PutMany(blocks))

	prs, err := bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	// Read lock will block the collect
	locker := gc.Locker()
	locker.Lock()

	collectCh := make(chan interface{})
	collectFinishCh := make(chan interface{})

	go func() {
		collectCh <- nil
		assert.Nil(gc.Collect())
		collectFinishCh <- nil
	}()

	// Wait the goutine to reach the collect wait
	<-collectCh

	prs, err = bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(true, prs)

	locker.Unlock()

	// wait gorountine to finish the gc collect call
	<-collectFinishCh

	prs, err = bs.Has(blocks[0].Cid())
	assert.Nil(err)
	assert.Equal(false, prs)

	prs, err = bs.Has(blocks[1].Cid())
	assert.Nil(err)
	assert.Equal(false, prs)
}

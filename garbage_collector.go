package crabfs

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	pb "github.com/runletapp/crabfs/protos"

	syncevent "github.com/GustavoKatel/SyncEvent"
	cid "github.com/ipfs/go-cid"
	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreQuery "github.com/ipfs/go-datastore/query"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/runletapp/crabfs/interfaces"
)

var _ interfaces.GarbageCollector = &garbageCollectorImpl{}

type garbageCollectorImpl struct {
	interval time.Duration

	scheduleCh chan interface{}

	ctx context.Context

	ds ipfsDatastore.Batching
	bs ipfsBlockstore.Blockstore

	locker *sync.RWMutex

	event syncevent.SyncEvent
}

// GarbageCollectorNew basic implementation of blockstore garbage collector
func GarbageCollectorNew(ctx context.Context, interval time.Duration, ds ipfsDatastore.Batching, bs ipfsBlockstore.Blockstore) (interfaces.GarbageCollector, error) {
	gc := &garbageCollectorImpl{
		ctx:      ctx,
		interval: interval,

		scheduleCh: make(chan interface{}, 1),

		ds: ds,
		bs: bs,

		locker: &sync.RWMutex{},
		event:  syncevent.NewSyncEvent(false),
	}

	return gc, nil
}

func (gc *garbageCollectorImpl) Start() error {
	go gc.background()

	return nil
}

func (gc *garbageCollectorImpl) Schedule() error {
	gc.scheduleCh <- nil

	return nil
}

func (gc *garbageCollectorImpl) background() {
	ticker := time.NewTicker(gc.interval)

	for {
		select {
		case <-gc.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			gc.Collect()
		case <-gc.scheduleCh:
			gc.Collect()
		}
	}
}

func (gc *garbageCollectorImpl) Collect() error {
	// Check if there's another collect operation going on
	if gc.event.IsSet() {
		return nil
	}

	gc.event.Set()
	defer gc.event.Reset()

	// Lock for read/write, making sure no other operation is happening during collecting
	gc.locker.Lock()
	defer gc.locker.Unlock()

	query := ipfsDatastoreQuery.Query{
		Prefix: "/crabfs/",
	}

	results, err := gc.ds.Query(query)
	if err != nil {
		return err
	}

	usedBlocks := map[cid.Cid]interface{}{}

	var record pb.DHTNameRecord
	var recordValue pb.DHTNameRecordValue
	for result := range results.Next() {
		if err := proto.Unmarshal(result.Value, &record); err != nil {
			continue
		}

		if err := proto.Unmarshal(record.Data, &recordValue); err != nil {
			continue
		}

		for _, blockMeta := range recordValue.Blocks {
			cid, _ := cid.Cast(blockMeta.Cid)
			usedBlocks[cid] = nil
		}
	}

	ch, err := gc.bs.AllKeysChan(gc.ctx)
	if err != nil {
		return err
	}

	for key := range ch {
		if _, prs := usedBlocks[key]; prs {
			// Block is used, next
			continue
		}

		err := gc.bs.DeleteBlock(key)
		if err != nil {
			return err
		}
	}

	gcDs, ok := gc.ds.(ipfsDatastore.GCDatastore)
	if ok {
		if err := gcDs.CollectGarbage(); err != nil {
			return err
		}
	}

	return nil
}

func (gc *garbageCollectorImpl) Locker() sync.Locker {
	return gc.locker.RLocker()
}

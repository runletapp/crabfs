package crabfs

import (
	"context"
	"crypto/rand"
	"io"
	"time"

	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"

	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreSync "github.com/ipfs/go-datastore/sync"
	ipfsDsBadger "github.com/ipfs/go-ds-badger"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
)

var _ interfaces.Core = &crabFS{}

type crabFS struct {
	settings *options.Settings

	host interfaces.Host

	datastore  *ipfsDatastoreSync.MutexDatastore
	blockstore ipfsBlockstore.Blockstore

	fetcherFactory interfaces.FetcherFactory

	gc interfaces.GarbageCollector
}

// New create a new CrabFS
func New(opts ...options.Option) (interfaces.Core, error) {
	settings := options.Settings{}
	if err := settings.SetDefaults(); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(&settings); err != nil {
			return nil, err
		}
	}

	if settings.BlockSize == 0 {
		return nil, ErrInvalidBlockSize
	}

	var privateKey *libp2pCrypto.RsaPrivateKey
	var err error
	if settings.PrivateKey == nil {
		privKey, _, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
		if err != nil {
			return nil, err
		}
		privateKey = privKey.(*libp2pCrypto.RsaPrivateKey)
	} else {
		privateKey, err = ReadPrivateKey(settings.PrivateKey)
		if err != nil {
			return nil, err
		}
	}

	var rawDatastore ipfsDatastore.Datastore
	if settings.Root == "" {
		// Use an in-memory datastore
		rawDatastore = ipfsDatastore.NewMapDatastore()
	} else {
		var err error
		rawDatastore, err = ipfsDsBadger.NewDatastore(settings.Root, nil)
		if err != nil {
			return nil, err
		}
	}

	datastore := ipfsDatastoreSync.MutexWrap(rawDatastore)

	blockstore := ipfsBlockstore.NewBlockstore(datastore)

	gc, err := GarbageCollectorNew(settings.Context, settings.GCInterval, datastore, blockstore)
	if err != nil {
		return nil, err
	}

	hostFactory := HostNew
	host, err := hostFactory(&settings, privateKey, datastore, blockstore)
	if err != nil {
		return nil, err
	}

	if !settings.RelayOnly {
		if err := host.Announce(); err != nil {
			return nil, err
		}
	}

	fs := &crabFS{
		settings: &settings,

		host: host,

		datastore: datastore,

		blockstore: blockstore,

		fetcherFactory: BasicFetcherNew,

		gc: gc,
	}

	if !settings.RelayOnly {
		go fs.background()

		if err := gc.Start(); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

func (fs *crabFS) background() {
	locker := fs.gc.Locker()
	locker.Lock()
	fs.host.Reprovide(fs.settings.Context)
	locker.Unlock()

	ticker := time.NewTicker(fs.settings.ReprovideInterval)
	for {
		select {
		case <-ticker.C:
			locker := fs.gc.Locker()
			locker.Lock()
			fs.host.Reprovide(fs.settings.Context)
			locker.Unlock()
		case <-fs.settings.Context.Done():
			return
		}
	}
}

func (fs *crabFS) Close() error {
	return fs.datastore.Close()
}

func (fs *crabFS) Get(ctx context.Context, bucket string, filename string) (interfaces.Fetcher, error) {
	locker := fs.gc.Locker()
	locker.Lock()

	blockMap, err := fs.host.GetContent(ctx, bucket, filename)
	if err != nil {
		locker.Unlock()
		return nil, err
	}

	fetcher, err := fs.fetcherFactory(ctx, fs, blockMap)
	if err != nil {
		locker.Unlock()
		return nil, err
	}

	go func() {
		// We hold a gc locker until the fetcher is done, this way we avoid
		// the gc messing around the fetch operation
		<-fetcher.Context().Done()
		locker.Unlock()
	}()

	return fetcher, nil
}

func (fs *crabFS) Put(ctx context.Context, bucket string, filename string, file io.Reader, mtime time.Time) error {
	locker := fs.gc.Locker()
	locker.Lock()
	defer locker.Unlock()

	slicer, err := BlockSlicerNew(file, fs.settings.BlockSize)
	if err != nil {
		return err
	}

	blockMap := interfaces.BlockMap{}
	totalSize := int64(0)

	blockMeta, block, err := slicer.Next()
	for {
		if err != nil && err != io.EOF {
			return err
		}

		if block == nil {
			break
		}

		if err := fs.blockstore.Put(block); err != nil {
			return err
		}

		blockMap[blockMeta.Start] = blockMeta
		totalSize += blockMeta.Size

		// Process block
		blockMeta, block, err = slicer.Next()
	}

	return fs.host.Publish(ctx, bucket, filename, blockMap, mtime, totalSize)
}

func (fs *crabFS) Remove(ctx context.Context, bucket string, filename string) error {
	if err := fs.gc.Schedule(); err != nil {
		return err
	}
	return fs.host.Remove(ctx, bucket, filename)
}

func (fs *crabFS) GetID() string {
	return fs.host.GetID()
}

func (fs *crabFS) GetAddrs() []string {
	return fs.host.GetAddrs()
}

func (fs *crabFS) Blockstore() ipfsBlockstore.Blockstore {
	return fs.blockstore
}

func (fs *crabFS) Host() interfaces.Host {
	return fs.host
}

func (fs *crabFS) GarbageCollector() interfaces.GarbageCollector {
	return fs.gc
}

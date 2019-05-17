package crabfs

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"

	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreSync "github.com/ipfs/go-datastore/sync"
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

	// TODO: Persistence
	datastore := ipfsDatastoreSync.MutexWrap(ipfsDatastore.NewMapDatastore())

	blockstore := ipfsBlockstore.NewBlockstore(datastore)

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
	}

	return fs, nil
}

func (fs *crabFS) Get(ctx context.Context, filename string) (io.ReadSeeker, int64, error) {
	blockMap, err := fs.host.GetContent(ctx, filename)
	if err != nil {
		return nil, 0, err
	}

	fetcher, err := fs.fetcherFactory(ctx, fs, blockMap)
	if err != nil {
		return nil, 0, err
	}

	return fetcher, fetcher.Size(), nil
}

func (fs *crabFS) Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error {
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

	return fs.host.Publish(ctx, filename, blockMap, mtime, totalSize)
}

func (fs *crabFS) Remove(ctx context.Context, filename string) error {
	blockMap, err := fs.host.GetContent(ctx, filename)
	if err != nil {
		return err
	}

	for _, blockMeta := range blockMap {
		cid, _ := cid.Cast(blockMeta.Cid)

		if err := fs.blockstore.DeleteBlock(cid); err != nil {
			fmt.Printf("Could not delete block: %s", cid.String())
		}
	}

	return nil
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

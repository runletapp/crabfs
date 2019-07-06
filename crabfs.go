package crabfs

import (
	"context"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/identity"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
)

var _ interfaces.Core = &crabFS{}

type crabFS struct {
	settings *options.Settings

	host interfaces.Host

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

	if settings.Identity == nil {
		var err error
		settings.Identity, err = identity.CreateIdentity()
		if err != nil {
			return nil, err
		}
	}

	hostFactory := HostNew
	host, err := hostFactory(&settings)
	if err != nil {
		return nil, err
	}

	fs := &crabFS{
		settings: &settings,

		host: host,

		fetcherFactory: BasicFetcherNew,
	}

	return fs, nil
}

func (fs *crabFS) Close() error {
	if err := fs.host.Close(); err != nil {
		return err
	}

	return nil
}

func (fs *crabFS) GetID() string {
	return fs.host.GetID()
}

func (fs *crabFS) GetAddrs() []string {
	return fs.host.GetAddrs()
}

func (fs *crabFS) Host() interfaces.Host {
	return fs.host
}

func (fs *crabFS) WithBucketFiles(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketAddr string) (interfaces.Bucket, error) {
	if err := fs.host.VerifyBucketSignature(ctx, privateKey, bucketAddr); err != nil {
		return nil, err
	}

	fs.host.SyncBucket(ctx, bucketAddr)

	return BucketCoreNew(fs, privateKey, bucketAddr, ""), nil
}

func (fs *crabFS) WithBucketFilesRoot(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucketAddr string, baseDir string) (interfaces.Bucket, error) {
	if err := fs.host.VerifyBucketSignature(ctx, privateKey, bucketAddr); err != nil {
		return nil, err
	}

	fs.host.SyncBucket(ctx, bucketAddr)

	return BucketCoreNew(fs, privateKey, bucketAddr, baseDir), nil
}

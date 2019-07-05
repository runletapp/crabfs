package crabfs

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"io"
	"time"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/identity"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"

	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
)

var _ interfaces.Core = &crabFS{}

type crabFS struct {
	settings *options.Settings

	host interfaces.Host

	addressBook interfaces.AddressBook

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

	addressBook, err := AddressBookNew()
	if err != nil {
		return nil, err
	}

	for bucket, addr := range settings.BucketAddress {
		if err := addressBook.Add(bucket, addr); err != nil {
			return nil, err
		}
	}

	hostFactory := HostNew
	host, err := hostFactory(&settings, addressBook)
	if err != nil {
		return nil, err
	}

	fs := &crabFS{
		settings: &settings,

		addressBook: addressBook,

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

func (fs *crabFS) Get(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (interfaces.Fetcher, error) {
	object, err := fs.host.GetContent(ctx, privateKey.GetPublic(), bucket, filename)
	if err != nil {
		return nil, err
	}

	fetcher, err := fs.fetcherFactory(ctx, fs, object, privateKey)
	if err != nil {
		return nil, err
	}

	return fetcher, nil
}

func (fs *crabFS) Lock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) (*pb.LockToken, error) {
	return fs.host.Lock(ctx, privateKey, bucket, filename)
}

func (fs *crabFS) Unlock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, token *pb.LockToken) error {
	return fs.host.Unlock(ctx, privateKey, bucket, filename, token)
}

func (fs *crabFS) IsLocked(ctx context.Context, publicKey crabfsCrypto.PubKey, bucket string, filename string) (bool, error) {
	return fs.host.IsLocked(ctx, publicKey, bucket, filename)
}

func (fs *crabFS) generateKey(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}

func (fs *crabFS) prepareFile(ctx context.Context, privateKey crabfsCrypto.PrivKey, file io.Reader) (interfaces.BlockMap, []byte, int64, error) {
	key, err := fs.generateKey(32) // aes-256
	if err != nil {
		return nil, nil, 0, err
	}

	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, 0, err
	}

	slicer, err := IPFSBlockSlicerNew(file, cipher)
	if err != nil {
		return nil, nil, 0, err
	}

	blockMap := interfaces.BlockMap{}
	totalSize := int64(0)

	blockMeta, block, n, err := slicer.Next()
	for {
		if err != nil && err != io.EOF {
			return nil, nil, 0, err
		}

		if block == nil {
			break
		}

		cid, err := fs.host.PutBlock(ctx, block)
		if err != nil {
			return nil, nil, 0, err
		}
		blockMeta.Cid = cid.Bytes()

		blockMap[blockMeta.Start] = blockMeta
		totalSize += n

		// Process block
		blockMeta, block, n, err = slicer.Next()
	}

	cipherKey, err := privateKey.GetPublic().Encrypt(key, []byte("crabfs"))
	if err != nil {
		return nil, nil, 0, err
	}

	return blockMap, cipherKey, totalSize, nil
}

func (fs *crabFS) Put(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, file io.Reader, mtime time.Time) error {
	blockMap, cipherKey, totalSize, err := fs.prepareFile(ctx, privateKey, file)
	if err != nil {
		return err
	}

	return fs.host.Publish(ctx, privateKey, cipherKey, bucket, filename, blockMap, mtime, totalSize)
}

func (fs *crabFS) PutWithCacheTTL(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, file io.Reader, mtime time.Time, ttl uint64) error {
	blockMap, cipherKey, totalSize, err := fs.prepareFile(ctx, privateKey, file)
	if err != nil {
		return err
	}

	return fs.host.PublishWithCacheTTL(ctx, privateKey, cipherKey, bucket, filename, blockMap, mtime, totalSize, ttl)
}

func (fs *crabFS) PutAndLock(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string, file io.Reader, mtime time.Time) (*pb.LockToken, error) {
	blockMap, cipherKey, totalSize, err := fs.prepareFile(ctx, privateKey, file)
	if err != nil {
		return nil, err
	}

	return fs.host.PublishAndLock(ctx, privateKey, cipherKey, bucket, filename, blockMap, mtime, totalSize)
}

func (fs *crabFS) Remove(ctx context.Context, privateKey crabfsCrypto.PrivKey, bucket string, filename string) error {
	return fs.host.Remove(ctx, privateKey, bucket, filename)
}

func (fs *crabFS) GetID() string {
	return fs.host.GetID()
}

func (fs *crabFS) GetAddrs() []string {
	return fs.host.GetAddrs()
}

func (fs *crabFS) Blockstore() ipfsBlockstore.Blockstore {
	return nil
}

func (fs *crabFS) Host() interfaces.Host {
	return fs.host
}

func (fs *crabFS) GarbageCollector() interfaces.GarbageCollector {
	return nil
}

func (fs *crabFS) WithBucket(privateKey crabfsCrypto.PrivKey, bucket string) (interfaces.Bucket, error) {
	if err := fs.PublishPublicKey(privateKey.GetPublic()); err != nil {
		return nil, err
	}

	return BucketCoreNew(fs, privateKey, bucket, ""), nil
}

func (fs *crabFS) WithBucketRoot(privateKey crabfsCrypto.PrivKey, bucket string, baseDir string) (interfaces.Bucket, error) {
	if err := fs.PublishPublicKey(privateKey.GetPublic()); err != nil {
		return nil, err
	}

	return BucketCoreNew(fs, privateKey, bucket, baseDir), nil
}

func (fs *crabFS) PublishPublicKey(publicKey crabfsCrypto.PubKey) error {
	return fs.host.PutPublicKey(publicKey)
}

func (fs *crabFS) GetIdentity() identity.Identity {
	return fs.settings.Identity
}

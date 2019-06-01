package crabfs

import (
	"context"
	"io"
	"path"
	"time"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"
)

var _ interfaces.Bucket = &bucketCoreImpl{}

type bucketCoreImpl struct {
	privateKey crabfsCrypto.PrivKey
	bucket     string

	fs interfaces.Core

	root string
}

// BucketCoreNew creates a new bucket core io
func BucketCoreNew(fs interfaces.Core, privateKey crabfsCrypto.PrivKey, bucket string, root string) interfaces.Bucket {
	return &bucketCoreImpl{
		privateKey: privateKey,
		bucket:     bucket,

		root: root,

		fs: fs,
	}
}

func (b *bucketCoreImpl) Get(ctx context.Context, filename string) (interfaces.Fetcher, error) {
	return b.fs.Get(ctx, b.privateKey, b.bucket, path.Join(b.root, filename))
}

func (b *bucketCoreImpl) Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error {
	return b.fs.Put(ctx, b.privateKey, b.bucket, path.Join(b.root, filename), file, mtime)
}

func (b *bucketCoreImpl) Remove(ctx context.Context, filename string) error {
	return b.fs.Remove(ctx, b.privateKey, b.bucket, path.Join(b.root, filename))
}

func (b *bucketCoreImpl) Chroot(dir string) interfaces.Bucket {
	return BucketCoreNew(b.fs, b.privateKey, b.bucket, path.Join(b.root, dir))
}

func (b *bucketCoreImpl) Lock(ctx context.Context, filename string) (*pb.LockToken, error) {
	return b.fs.Lock(ctx, b.privateKey, b.bucket, path.Join(b.root, filename))
}

func (b *bucketCoreImpl) Unlock(ctx context.Context, filename string, token *pb.LockToken) error {
	return b.fs.Unlock(ctx, b.privateKey, b.bucket, path.Join(b.root, filename), token)
}

func (b *bucketCoreImpl) IsLocked(ctx context.Context, filename string) (bool, error) {
	return b.fs.IsLocked(ctx, b.privateKey.GetPublic(), b.bucket, path.Join(b.root, filename))
}

package crabfs

import (
	"context"
	"io"
	"time"

	"github.com/runletapp/crabfs/interfaces"
)

var _ interfaces.Bucket = &bucketCoreImpl{}

type bucketCoreImpl struct {
	privateKey interfaces.PrivKey
	bucket     string

	fs interfaces.Core
}

// BucketCoreNew creates a new bucket core io
func BucketCoreNew(fs interfaces.Core, privateKey interfaces.PrivKey, bucket string) interfaces.Bucket {
	return &bucketCoreImpl{
		privateKey: privateKey,
		bucket:     bucket,

		fs: fs,
	}
}

func (b *bucketCoreImpl) Get(ctx context.Context, filename string) (interfaces.Fetcher, error) {
	return b.fs.Get(ctx, b.privateKey.GetPublic(), b.bucket, filename)
}

func (b *bucketCoreImpl) Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error {
	return b.fs.Put(ctx, b.privateKey, b.bucket, filename, file, mtime)
}

func (b *bucketCoreImpl) Remove(ctx context.Context, filename string) error {
	return b.fs.Remove(ctx, b.privateKey, b.bucket, filename)
}

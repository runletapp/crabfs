package crabfs

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/runletapp/crabfs/mocks"
	pb "github.com/runletapp/crabfs/protos"

	"github.com/stretchr/testify/assert"
)

func TestBucketGet(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	fetcher := mocks.NewMockFetcher(ctrl)
	fs.EXPECT().Get(ctx, privKey, "bkt", "/root/test.txt").Return(fetcher, nil)

	f, err := bucket.Get(ctx, "test.txt")
	assert.Nil(err)
	assert.Equal(fetcher, f)
}

func TestBucketPut(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	reader := bytes.NewReader([]byte{})
	now := time.Now()
	fs.EXPECT().Put(ctx, privKey, "bkt", "/root/test.txt", reader, now).Return(nil)

	err = bucket.Put(ctx, "test.txt", reader, now)
	assert.Nil(err)
}

func TestBucketPutAndLock(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	reader := bytes.NewReader([]byte{})
	now := time.Now()
	lock := &pb.LockToken{}
	fs.EXPECT().PutAndLock(ctx, privKey, "bkt", "/root/test.txt", reader, now).Return(lock, nil)

	rlock, err := bucket.PutAndLock(ctx, "test.txt", reader, now)
	assert.Nil(err)
	assert.Equal(lock, rlock)
}

func TestBucketPutWithCacheTTL(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	reader := bytes.NewReader([]byte{})
	now := time.Now()
	ttl := uint64(777)
	fs.EXPECT().PutWithCacheTTL(ctx, privKey, "bkt", "/root/test.txt", reader, now, ttl).Return(nil)

	err = bucket.PutWithCacheTTL(ctx, "test.txt", reader, now, ttl)
	assert.Nil(err)
}

func TestBucketRemove(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	fs.EXPECT().Remove(ctx, privKey, "bkt", "/root/test.txt").Return(nil)

	err = bucket.Remove(ctx, "test.txt")
	assert.Nil(err)
}

func TestBucketBucketChroot(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	bucket = bucket.Chroot("path2")

	ctx := context.Background()

	fs.EXPECT().Remove(ctx, privKey, "bkt", "/root/path2/test.txt").Return(nil)

	err = bucket.Remove(ctx, "test.txt")
	assert.Nil(err)
}

func TestBucketLock(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	lock := &pb.LockToken{}

	fs.EXPECT().Lock(ctx, privKey, "bkt", "/root/test.txt").Return(lock, nil)

	rlock, err := bucket.Lock(ctx, "test.txt")
	assert.Nil(err)
	assert.Equal(lock, rlock)
}

func TestBucketUnlock(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	lock := &pb.LockToken{}

	fs.EXPECT().Unlock(ctx, privKey, "bkt", "/root/test.txt", lock).Return(nil)

	err = bucket.Unlock(ctx, "test.txt", lock)
	assert.Nil(err)
}

func TestBucketIsLocked(t *testing.T) {
	ctrl := setUpBasicTest(t)
	defer setDownBasicTest(ctrl)

	assert := assert.New(t)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	fs := mocks.NewMockCore(ctrl)

	bucket := BucketCoreNew(fs, privKey, "bkt", "/root")

	ctx := context.Background()

	fs.EXPECT().IsLocked(ctx, privKey.GetPublic(), "bkt", "/root/test.txt").Return(true, nil)

	ret, err := bucket.IsLocked(ctx, "test.txt")
	assert.Nil(err)
	assert.True(ret)
}

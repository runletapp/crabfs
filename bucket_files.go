package crabfs

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/golang/protobuf/proto"
	multihash "github.com/multiformats/go-multihash"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"
)

var _ interfaces.Bucket = &bucketCoreImpl{}

type bucketCoreImpl struct {
	privateKey    crabfsCrypto.PrivKey
	bucketAddress string

	fs interfaces.Core

	root string
}

// BucketCoreNew creates a new bucket core io
func BucketCoreNew(fs interfaces.Core, privateKey crabfsCrypto.PrivKey, bucketAddress string, root string) interfaces.Bucket {
	return &bucketCoreImpl{
		privateKey:    privateKey,
		bucketAddress: bucketAddress,

		root: root,

		fs: fs,
	}
}

func (b *bucketCoreImpl) generateKey(size int) ([]byte, error) {
	buff := make([]byte, size)
	_, err := rand.Read(buff)
	return buff, err
}

func (b *bucketCoreImpl) prepareFile(ctx context.Context, privateKey crabfsCrypto.PrivKey, file io.Reader) (interfaces.BlockMap, []byte, int64, error) {
	key, err := b.generateKey(32) // aes-256
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

		cid, err := b.fs.Host().PutBlock(ctx, block)
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

func (b *bucketCoreImpl) Get(ctx context.Context, filename string) (interfaces.Fetcher, error) {
	// return b.fs.Get(ctx, b.privateKey, b.bucket, path.Join(b.root, filename))
	return nil, fmt.Errorf("TODO")
}

func (b *bucketCoreImpl) keyFromFilename(filename string) string {
	hash, _ := multihash.Sum([]byte(filename), multihash.SHA3_256, -1)
	return fmt.Sprintf("/crabfs/v1/%s", hash.String())
}

func (b *bucketCoreImpl) publishObject(ctx context.Context, object *pb.CrabObject) error {
	entry := &pb.CrabEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	data, err := proto.Marshal(object)
	if err != nil {
		return err
	}

	entry.Data = data

	return b.fs.Host().BucketAppend(ctx, b.privateKey, b.bucketAddress, entry)
}

func (b *bucketCoreImpl) Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error {
	blockMap, cipherKey, totalSize, err := b.prepareFile(ctx, b.privateKey, file)
	if err != nil {
		return err
	}

	key := b.keyFromFilename(path.Join(b.root, filename))

	object := &pb.CrabObject{
		Name:   key,
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   totalSize,
		Key:    cipherKey,
		Delete: false,
		Lock:   nil,
	}

	return b.publishObject(ctx, object)
}

func (b *bucketCoreImpl) Remove(ctx context.Context, filename string) error {
	key := b.keyFromFilename(path.Join(b.root, filename))

	object := &pb.CrabObject{
		Name:   key,
		Blocks: map[int64]*pb.BlockMetadata{},
		Mtime:  time.Now().UTC().Format(time.RFC3339Nano),
		Size:   0,
		Key:    []byte{},
		Delete: true,
		Lock:   nil,
	}

	return b.publishObject(ctx, object)
}

func (b *bucketCoreImpl) Chroot(dir string) interfaces.Bucket {
	return BucketCoreNew(b.fs, b.privateKey, b.bucketAddress, path.Join(b.root, dir))
}

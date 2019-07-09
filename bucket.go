package crabfs

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/multiformats/go-multihash"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"

	ipfsLog "berty.tech/go-ipfs-log"
	ipfsLogEntry "berty.tech/go-ipfs-log/entry"
	"github.com/ipfs/go-cid"
	ipfsPath "github.com/ipfs/interface-go-ipfs-core/path"
)

var _ interfaces.Bucket = (*bucketImpl)(nil)

type bucketImpl struct {
	fs  interfaces.Core
	log *ipfsLog.Log

	addr    string
	privKey crabfsCrypto.PrivKey
}

// NewBucketRoot creates a bucket root
func NewBucketRoot(ctx context.Context, id string) ([]byte, error) {
	object := pb.CrabObject{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		BucketID:  id,
	}

	return proto.Marshal(&object)
}

// NewBucket create a new fs bucket from a base bucket
func NewBucket(fs interfaces.Core, log *ipfsLog.Log, addr string, privKey crabfsCrypto.PrivKey) (interfaces.Bucket, error) {
	b := &bucketImpl{
		fs:      fs,
		log:     log,
		addr:    addr,
		privKey: privKey,
	}

	return b, nil
}

func (bucket *bucketImpl) Address() string {
	return bucket.addr
}

func (bucket *bucketImpl) keyFromFilename(filename string) string {
	hash, _ := multihash.Sum([]byte(filename), multihash.SHA3_256, -1)
	return fmt.Sprintf("/crabfs/v1/%s", hash.String())
}

func (bucket *bucketImpl) putBlock(ctx context.Context, data []byte) (cid.Cid, error) {
	nd, err := bucket.fs.Storage().Object().New(ctx)
	if err != nil {
		return cid.Cid{}, err
	}

	path, err := bucket.fs.Storage().Object().SetData(ctx, ipfsPath.IpfsPath(nd.Cid()), bytes.NewReader(data))
	if err != nil {
		return cid.Cid{}, err
	}

	return path.Cid(), nil
}

func (bucket *bucketImpl) generateKey(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}

func (bucket *bucketImpl) prepareFile(ctx context.Context, privateKey crabfsCrypto.PrivKey, file io.Reader) (interfaces.BlockMap, []byte, int64, error) {
	key, err := bucket.generateKey(32) // aes-256
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

	blockMeta, blockData, n, err := slicer.Next()
	for {
		if err != nil && err != io.EOF {
			return nil, nil, 0, err
		}

		if blockData == nil {
			break
		}

		cid, err := bucket.putBlock(ctx, blockData)
		if err != nil {
			return nil, nil, 0, err
		}
		blockMeta.Cid = cid.Bytes()

		blockMap[blockMeta.Start] = blockMeta
		totalSize += n

		// Process block
		blockMeta, blockData, n, err = slicer.Next()
	}

	cipherKey, err := privateKey.GetPublic().Encrypt(key, []byte("crabfs"))
	if err != nil {
		return nil, nil, 0, err
	}

	return blockMap, cipherKey, totalSize, nil
}

func (bucket *bucketImpl) publishObject(ctx context.Context, object *pb.CrabObject) error {
	data, err := proto.Marshal(object)
	if err != nil {
		return err
	}

	_, err = bucket.log.Append(ctx, data, 1)
	return err
}

func (bucket *bucketImpl) Put(ctx context.Context, filename string, r io.Reader, mtime time.Time) error {
	blockMap, cipherKey, totalSize, err := bucket.prepareFile(ctx, bucket.privKey, r)
	if err != nil {
		return err
	}

	key := bucket.keyFromFilename(filename)

	object := &pb.CrabObject{
		Name:   key,
		Blocks: blockMap,
		Mtime:  mtime.UTC().Format(time.RFC3339Nano),
		Size:   totalSize,
		Key:    cipherKey,
		Delete: false,
		Lock:   nil,
	}

	return bucket.publishObject(ctx, object)
}

func (bucket *bucketImpl) xObjectFromEntry(entry *ipfsLogEntry.Entry) (*pb.CrabObject, error) {
	var obj pb.CrabObject
	if err := proto.Unmarshal(entry.Payload, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}

func (bucket *bucketImpl) convergeKey(ctx context.Context, key string) (*pb.CrabObject, error) {

	var objectConverged *pb.CrabObject

	for _, entry := range bucket.log.Values().Slice() {
		object, err := bucket.xObjectFromEntry(entry)
		if err != nil {
			continue
		}

		if strings.Compare(object.Name, key) == 0 {
			objectConverged = object
		}
	}

	if objectConverged == nil {
		return nil, ErrObjectNotFound
	}

	return objectConverged, nil
}

func (bucket *bucketImpl) Get(ctx context.Context, filename string) (interfaces.Fetcher, error) {

	object, err := bucket.convergeKey(ctx, bucket.keyFromFilename(filename))
	if err != nil {
		return nil, err
	}

	return BasicFetcherNew(ctx, bucket.fs, object, bucket.privKey)
}

func (bucket *bucketImpl) Remove(ctx context.Context, filename string) error {
	key := bucket.keyFromFilename(filename)

	object := &pb.CrabObject{
		Name:   key,
		Blocks: map[int64]*pb.BlockMetadata{},
		Mtime:  time.Now().UTC().Format(time.RFC3339Nano),
		Size:   0,
		Key:    []byte{},
		Delete: true,
		Lock:   nil,
	}

	return bucket.publishObject(ctx, object)
}

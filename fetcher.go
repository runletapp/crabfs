package crabfs

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"io/ioutil"
	"os"
	"sort"

	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"

	"github.com/ipfs/go-cid"
	ipfsPath "github.com/ipfs/interface-go-ipfs-core/path"
)

var _ interfaces.Fetcher = &BasicFetcher{}

// BasicFetcher single peer block fetcher
type BasicFetcher struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	offset int64

	keys []int64

	buffer *bytes.Buffer

	totalSize int64

	blockMap interfaces.BlockMap

	fs interfaces.Core

	privateKey crabfsCrypto.PrivKey

	cipher cipher.Block
}

// BasicFetcherNew creates a new basic fetcher
func BasicFetcherNew(ctx context.Context, fs interfaces.Core, object *pb.CrabObject, privateKey crabfsCrypto.PrivKey) (interfaces.Fetcher, error) {
	keys := []int64{}

	blockMap := object.Blocks

	for key := range blockMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	key, err := privateKey.Decrypt(object.Key, []byte("crabfs"))
	if err != nil {
		return nil, err
	}
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	fetcher := &BasicFetcher{
		ctx:       ctx,
		ctxCancel: cancel,

		offset: 0,
		keys:   keys,

		buffer: &bytes.Buffer{},

		totalSize: object.Size,

		blockMap: blockMap,

		fs: fs,

		privateKey: privateKey,

		cipher: cipher,
	}

	return fetcher, nil
}

func (fetcher *BasicFetcher) Size() int64 {
	return fetcher.totalSize
}

func (fetcher *BasicFetcher) getNextIndex(offset int64) int64 {
	currentIndex := int64(0)
	for _, index := range fetcher.keys {
		if index > offset {
			break
		}

		currentIndex = index
	}

	return currentIndex
}

func (fetcher *BasicFetcher) getDataFromBlock(meta *pb.BlockMetadata, data []byte) ([]byte, error) {
	fetcher.cipher.Decrypt(data, data)

	return data[:int(meta.PaddingStart)], nil
}

func (fetcher *BasicFetcher) Read(p []byte) (n int, err error) {
	if fetcher.offset >= fetcher.totalSize {
		return 0, io.EOF
	}

	limit := fetcher.totalSize - fetcher.offset
	if int64(len(p)) < limit {
		limit = int64(len(p))
	}

	// pLimit := p[:limit]
	pLimit := p

	if fetcher.buffer.Len() > 0 {
		n, err := fetcher.buffer.Read(pLimit)
		if err != nil {
			return 0, err
		}

		fetcher.offset += int64(n)

		return n, nil
	}

	nextIndex := fetcher.getNextIndex(fetcher.offset)
	blockMeta, prs := fetcher.blockMap[nextIndex]

	if !prs {
		return 0, ErrInvalidOffset
	}

	localOffset := int64(0)
	if fetcher.offset > blockMeta.Start {
		localOffset = fetcher.offset - blockMeta.Start
	}

	// TODO: improvement: fetch-ahead blocks
	block, err := fetcher.downloadBlock(blockMeta)
	if err != nil {
		return 0, err
	}

	plainData, err := fetcher.getDataFromBlock(blockMeta, block)
	if err != nil {
		return 0, err
	}

	fetcher.buffer.Write(plainData[localOffset:])
	n, err = fetcher.buffer.Read(pLimit)
	if err != nil {
		return 0, err
	}

	fetcher.offset += int64(n)
	return n, nil
}

func (fetcher *BasicFetcher) downloadBlock(blockMeta *pb.BlockMetadata) ([]byte, error) {
	cid, err := cid.Cast(blockMeta.Cid)
	if err != nil {
		return nil, err
	}

	r, err := fetcher.fs.Storage().Object().Data(fetcher.ctx, ipfsPath.IpfsPath(cid))
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (fetcher *BasicFetcher) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case os.SEEK_END:
		fetcher.offset = fetcher.totalSize - offset
	case os.SEEK_CUR:
		fetcher.offset += offset
	case os.SEEK_SET:
		fetcher.offset = offset
	}

	if fetcher.offset < 0 || fetcher.offset > fetcher.totalSize {
		return 0, ErrInvalidOffset
	}

	// Clear the internal buffer
	fetcher.buffer.Reset()

	return fetcher.offset, nil
}

func (fetcher *BasicFetcher) Close() error {
	fetcher.ctxCancel()
	return nil
}

func (fetcher *BasicFetcher) Context() context.Context {
	return fetcher.ctx
}

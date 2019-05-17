package crabfs

import (
	"bytes"
	"context"
	"io"
	"os"
	"sort"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"
)

var _ interfaces.Fetcher = &BasicFetcher{}

// BasicFetcher single peer block fetcher
type BasicFetcher struct {
	ctx context.Context

	offset int64

	keys []int64

	buffer *bytes.Buffer

	totalSize int64

	blockMap interfaces.BlockMap

	fs interfaces.Core
}

// BasicFetcherNew creates a new basic fetcher
func BasicFetcherNew(ctx context.Context, fs interfaces.Core, blockMap interfaces.BlockMap) (interfaces.Fetcher, error) {
	keys := []int64{}

	totalSize := int64(0)

	for key, blockMeta := range blockMap {
		totalSize += blockMeta.Size
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	fetcher := &BasicFetcher{
		ctx: ctx,

		offset: 0,
		keys:   keys,

		buffer: &bytes.Buffer{},

		totalSize: totalSize,

		blockMap: blockMap,

		fs: fs,
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

func (fetcher *BasicFetcher) Read(p []byte) (n int, err error) {
	if fetcher.offset >= fetcher.totalSize {
		return 0, io.EOF
	}

	if fetcher.buffer.Len() >= cap(p) {
		n, err := fetcher.buffer.Read(p)
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

	cid, _ := cid.Cast(blockMeta.Cid)
	block, err := fetcher.fs.Blockstore().Get(cid)
	if err == nil {
		fetcher.buffer.Write(block.RawData()[localOffset:])

		n, err := fetcher.buffer.Read(p)
		if err != nil {
			return 0, err
		}

		fetcher.offset += int64(n)

		return n, err
	}

	// TODO: improvement: fetch-ahead blocks
	block, err = fetcher.downloadBlock(blockMeta)
	if err != nil {
		return 0, err
	}

	fetcher.buffer.Write(block.RawData()[localOffset:])
	n, err = fetcher.buffer.Read(p)
	if err != nil {
		return 0, err
	}

	fetcher.offset += int64(n)
	return n, nil
}

func (fetcher *BasicFetcher) downloadBlock(blockMeta *pb.BlockMetadata) (blocks.Block, error) {
	ctx, cancel := context.WithCancel(fetcher.ctx)
	defer cancel()

	ch := fetcher.fs.Host().FindProviders(ctx, blockMeta)
	for peer := range ch {

		buffer := &bytes.Buffer{}

		stream, err := fetcher.fs.Host().CreateBlockStream(fetcher.ctx, blockMeta, &peer)
		if err != nil {
			// return nil, err
			// Could not create stream, try the next peer
			continue
		}

		_, err = io.CopyN(buffer, stream, blockMeta.Size)
		if err != nil {
			// return nil, err
			// Short read, try the next peer
			continue
		}

		block := blocks.NewBlock(buffer.Bytes())

		cid, _ := cid.Cast(blockMeta.Cid)
		if !block.Cid().Equals(cid) {
			// We have received invalid data, try the next peer
			continue
		}

		if err := fetcher.fs.Blockstore().Put(block); err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, ErrBlockNotFound
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

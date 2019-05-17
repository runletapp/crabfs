package crabfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreSync "github.com/ipfs/go-datastore/sync"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/runletapp/crabfs/mocks"
	pb "github.com/runletapp/crabfs/protos"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	"github.com/runletapp/crabfs/interfaces"

	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
)

func setUpFetcherTest(t *testing.T, blockMap interfaces.BlockMap) (interfaces.Fetcher, interfaces.Core, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	fs := mocks.NewMockCore(ctrl)
	datastore := ipfsDatastoreSync.MutexWrap(ipfsDatastore.NewMapDatastore())
	blockstore := ipfsBlockstore.NewBlockstore(datastore)
	fs.EXPECT().Blockstore().AnyTimes().Return(blockstore)

	fetcher, err := BasicFetcherNew(context.Background(), fs, blockMap)
	assert.Nil(err)

	return fetcher, fs, ctrl
}

func setDownFetcherTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestFetcherSizeCalc(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  1,
		Cid:   []byte{},
	}
	blockMap[5] = &pb.BlockMetadata{
		Start: 5,
		Size:  9,
		Cid:   []byte{},
	}

	fetcher, _, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())
}

func TestFetcherReadWholeBlock(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 1)
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader(block.RawData()), nil)

	data := make([]byte, 100)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(10, n)
}

func TestFetcherReadOverCap(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("012345678901234567890123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(30), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 1)
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader(block.RawData()), nil)

	data := make([]byte, 2)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("01", string(data))

	n, err = fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("23", string(data))
}

func TestFetcherReadMidBlock(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 1)
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader(block.RawData()), nil)

	data := make([]byte, 2)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("01", string(data))

	// Seek will clear the internal buffer
	pos, err := fetcher.Seek(2, os.SEEK_CUR)
	assert.Nil(err)
	assert.Equal(int64(4), pos)

	// Our block should already be in the blockstore
	prs, err := fs.Blockstore().Has(block.Cid())
	assert.Nil(err)
	assert.True(prs)

	n, err = fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("45", string(data))
}

func TestFetcherSeekOverflow(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	assert.Nil(fs.Blockstore().Put(block))

	_, err := fetcher.Seek(1000000, os.SEEK_SET)
	assert.NotNil(err)
}

func TestFetcherSeekOverflowCur(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	assert.Nil(fs.Blockstore().Put(block))

	n, err := fetcher.Seek(8, os.SEEK_SET)
	assert.Equal(int64(8), n)
	assert.Nil(err)

	_, err = fetcher.Seek(3, os.SEEK_CUR)
	assert.NotNil(err)
}

func TestFetcherSeekEnd(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	assert.Nil(fs.Blockstore().Put(block))

	n, err := fetcher.Seek(fetcher.Size(), os.SEEK_SET)
	assert.Equal(fetcher.Size(), n)
	assert.Nil(err)

	data := make([]byte, 2)
	nRead, err := fetcher.Read(data)
	assert.Equal(io.EOF, err)
	assert.Equal(0, nRead)
}

func TestFetcherSeekEndBackwards(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	assert.Nil(fs.Blockstore().Put(block))

	n, err := fetcher.Seek(0, os.SEEK_END)
	assert.Equal(fetcher.Size(), n)
	assert.Nil(err)

	data := make([]byte, 2)
	nRead, err := fetcher.Read(data)
	assert.Equal(io.EOF, err)
	assert.Equal(0, nRead)
}

func TestFetcherSeekCurEnd(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	assert.Nil(fs.Blockstore().Put(block))

	n, err := fetcher.Seek(fetcher.Size(), os.SEEK_CUR)
	assert.Equal(fetcher.Size(), n)
	assert.Nil(err)

	data := make([]byte, 2)
	nRead, err := fetcher.Read(data)
	assert.Equal(io.EOF, err)
	assert.Equal(0, nRead)
}

func TestFetcherDownloadInvalidData(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	peerCh <- libp2pPeerstore.PeerInfo{}
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	// We try two peers. First one has invalid data
	// the second one has correct data
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader([]byte("1123456789")), nil)
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader(block.RawData()), nil)

	data := make([]byte, 2)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("01", string(data))
}

func TestFetcherDownloadShortRead(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	peerCh <- libp2pPeerstore.PeerInfo{}
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	// We try two peers. First one has invalid data
	// the second one has correct data
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader([]byte("0123")), nil)
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader(block.RawData()), nil)

	data := make([]byte, 2)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("01", string(data))
}

func TestFetcherDownloadErrorCreateStream(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	peerCh <- libp2pPeerstore.PeerInfo{}
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	// We try two peers. First one has invalid data
	// the second one has correct data
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("Error Test"))
	host.EXPECT().CreateBlockStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(bytes.NewReader(block.RawData()), nil)

	data := make([]byte, 2)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("01", string(data))
}

func TestFetcherDownloadNotFound(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block := CreateBlockFromString("0123456789")
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	fetcher, fs, ctrl := setUpFetcherTest(t, blockMap)
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	close(peerCh)

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	data := make([]byte, 2)
	_, err := fetcher.Read(data)
	assert.Equal(ErrBlockNotFound, err)
}

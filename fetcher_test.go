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
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/mocks"
	pb "github.com/runletapp/crabfs/protos"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	"github.com/runletapp/crabfs/interfaces"

	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
)

func setUpFetcherTest(t *testing.T, object *pb.CrabObject) (interfaces.Fetcher, interfaces.Core, crabfsCrypto.PrivKey, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	fs := mocks.NewMockCore(ctrl)
	datastore := ipfsDatastoreSync.MutexWrap(ipfsDatastore.NewMapDatastore())
	blockstore := ipfsBlockstore.NewBlockstore(datastore)
	fs.EXPECT().Blockstore().AnyTimes().Return(blockstore)

	privateKey, err := GenerateKeyPair()
	assert.Nil(err)

	key, err := privateKey.GetPublic().Encrypt([]byte("0123456789abcdef"), []byte("crabfs"))
	assert.Nil(err)

	object.Key = key

	fetcher, err := BasicFetcherNew(context.Background(), fs, object, privateKey)
	assert.Nil(err)

	return fetcher, fs, privateKey, ctrl
}

func setDownFetcherTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestFetcherSize(t *testing.T) {
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

	fetcher, _, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 55})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(55), fetcher.Size())
}

func TestFetcherReadWholeBlock(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), block.Cid())

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

	block, padding := CreateBlockFromString("012345678901234567890123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 30})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(30), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), block.Cid())

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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)

	peerCh := make(chan libp2pPeerstore.PeerInfo, 1)
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().Provide(gomock.Any(), block.Cid())
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

func TestFetcherReadBetweenBlocks(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block0, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block0.RawData())),
		Cid:          block0.Cid().Bytes(),
		PaddingStart: padding,
	}

	block10, padding := CreateBlockFromString("9876543210", []byte("0123456789abcdef"))
	blockMap[10] = &pb.BlockMetadata{
		Start:        10,
		Size:         int64(len(block10.RawData())),
		Cid:          block10.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 20})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(20), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), gomock.Any()).AnyTimes()

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	peerCh <- libp2pPeerstore.PeerInfo{}
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))
	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	host.EXPECT().CreateBlockStream(gomock.Any(), blockMap[0], gomock.Any()).Return(bytes.NewReader(block0.RawData()), nil)
	host.EXPECT().CreateBlockStream(gomock.Any(), blockMap[10], gomock.Any()).Return(bytes.NewReader(block10.RawData()), nil)

	data := make([]byte, 4)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(4, n)
	assert.Equal("0123", string(data))

	// Seek will clear the internal buffer
	pos, err := fetcher.Seek(8, os.SEEK_SET)
	assert.Nil(err)
	assert.Equal(int64(8), pos)

	n, err = io.ReadFull(fetcher, data)
	assert.Nil(err)
	assert.Equal(4, n)
	assert.Equal("8998", string(data))
}

func TestFetcherReadFinalBlock(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block0, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block0.RawData())),
		Cid:          block0.Cid().Bytes(),
		PaddingStart: padding,
	}

	block10, padding := CreateBlockFromString("987654321A", []byte("0123456789abcdef"))
	blockMap[10] = &pb.BlockMetadata{
		Start:        10,
		Size:         int64(len(block10.RawData())),
		Cid:          block10.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 20})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(20), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), gomock.Any()).AnyTimes()

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	peerCh <- libp2pPeerstore.PeerInfo{}
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))
	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	host.EXPECT().CreateBlockStream(gomock.Any(), blockMap[0], gomock.Any()).Return(bytes.NewReader(block0.RawData()), nil)
	host.EXPECT().CreateBlockStream(gomock.Any(), blockMap[10], gomock.Any()).Return(bytes.NewReader(block10.RawData()), nil)

	data := make([]byte, 4)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(4, n)
	assert.Equal("0123", string(data))

	// Seek will clear the internal buffer
	pos, err := fetcher.Seek(19, os.SEEK_SET)
	assert.Nil(err)
	assert.Equal(int64(19), pos)

	n, err = fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(1, n)
	assert.Equal("A", string(data[:n]))
}

func TestFetcherBigBlocks(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block0, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block0.RawData())),
		Cid:          block0.Cid().Bytes(),
		PaddingStart: padding,
	}

	block10, padding := CreateBlockFromString("98765", []byte("0123456789abcdef"))
	blockMap[10] = &pb.BlockMetadata{
		Start:        10,
		Size:         int64(len(block10.RawData())),
		Cid:          block10.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 15})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(15), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), gomock.Any()).AnyTimes()

	peerCh := make(chan libp2pPeerstore.PeerInfo, 2)
	peerCh <- libp2pPeerstore.PeerInfo{}
	peerCh <- libp2pPeerstore.PeerInfo{}

	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))
	host.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return((<-chan libp2pPeerstore.PeerInfo)(peerCh))

	host.EXPECT().CreateBlockStream(gomock.Any(), blockMap[0], gomock.Any()).Return(bytes.NewReader(block0.RawData()), nil)
	host.EXPECT().CreateBlockStream(gomock.Any(), blockMap[10], gomock.Any()).Return(bytes.NewReader(block10.RawData()), nil)

	data := make([]byte, 8)
	n, err := fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(8, n)
	assert.Equal("01234567", string(data))

	n, err = fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(2, n)
	assert.Equal("89", string(data[:n]))

	n, err = fetcher.Read(data)
	assert.Nil(err)
	assert.Equal(5, n)
	assert.Equal("98765", string(data[:n]))
}

func TestFetcherSeekOverflow(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	assert.Nil(fs.Blockstore().Put(block))

	_, err := fetcher.Seek(1000000, os.SEEK_SET)
	assert.NotNil(err)
}

func TestFetcherSeekOverflowCur(t *testing.T) {
	assert := assert.New(t)

	blockMap := interfaces.BlockMap{}

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), gomock.Any()).AnyTimes()

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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), gomock.Any()).AnyTimes()

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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
	defer setDownFetcherTest(ctrl)

	assert.Equal(int64(10), fetcher.Size())

	host := mocks.NewMockHost(ctrl)
	fs.(*mocks.MockCore).EXPECT().Host().AnyTimes().Return(host)
	host.EXPECT().Provide(gomock.Any(), gomock.Any()).AnyTimes()

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

	block, padding := CreateBlockFromString("0123456789", []byte("0123456789abcdef"))
	blockMap[0] = &pb.BlockMetadata{
		Start:        0,
		Size:         int64(len(block.RawData())),
		Cid:          block.Cid().Bytes(),
		PaddingStart: padding,
	}

	fetcher, fs, _, ctrl := setUpFetcherTest(t, &pb.CrabObject{Blocks: blockMap, Size: 10})
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

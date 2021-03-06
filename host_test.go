package crabfs

import (
	"context"
	"testing"
	"time"

	"github.com/runletapp/crabfs/identity"

	"github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"
	"github.com/stretchr/testify/assert"
)

func setUpHostBasic(t *testing.T, settings *options.Settings, ds ipfsDatastore.Batching, bs ipfsBlockstore.Blockstore) interfaces.Host {
	assert := assert.New(t)

	if settings == nil {
		settings = &options.Settings{}
	}

	if settings.Context == nil {
		settings.Context = context.Background()
	}

	if settings.Identity == nil {
		id, err := identity.CreateIdentity()
		assert.Nil(err)
		settings.Identity = id
	}

	if ds == nil {
		ds = ipfsDatastore.NewMapDatastore()
	}

	if bs == nil {
		bs = ipfsBlockstore.NewBlockstore(ds)
	}

	host, err := HostNew(settings, ds, bs)
	assert.Nil(err)

	return host
}

func setUpHostTestWithDsAndBs(t *testing.T) (interfaces.Host, ipfsDatastore.Datastore, ipfsBlockstore.Blockstore, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	id, err := identity.CreateIdentity()
	assert.Nil(err)

	settings := &options.Settings{
		Context:  context.Background(),
		Identity: id,
	}

	ds := ipfsDatastore.NewMapDatastore()
	bs := ipfsBlockstore.NewBlockstore(ds)

	host, err := HostNew(settings, ds, bs)
	assert.Nil(err)

	return host, ds, bs, ctrl
}

func setUpHostTest(t *testing.T) (interfaces.Host, *gomock.Controller) {
	host, _, _, ctrl := setUpHostTestWithDsAndBs(t)
	return host, ctrl
}

func setUpHostTestWithRelay(t *testing.T) (interfaces.Host, ipfsDatastore.Datastore, ipfsBlockstore.Blockstore, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	assert := assert.New(t)

	id, err := identity.CreateIdentity()
	assert.Nil(err)

	settings := &options.Settings{
		Context:  context.Background(),
		Identity: id,
	}

	ds := ipfsDatastore.NewMapDatastore()
	bs := ipfsBlockstore.NewBlockstore(ds)

	relay, err := RelayNew(settings.Context, 0, []string{}, nil)
	assert.Nil(err)

	settings.BootstrapPeers = relay.GetAddrs()

	host, err := HostNew(settings, ds, bs)
	assert.Nil(err)

	return host, ds, bs, ctrl
}

func setDownHostTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestHostValidID(t *testing.T) {
	assert := assert.New(t)
	host, ctrl := setUpHostTest(t)
	defer setDownHostTest(ctrl)

	assert.NotEqual("", host.GetID())
}

func TestHostAnnounceNoPeers(t *testing.T) {
	assert := assert.New(t)
	host, ctrl := setUpHostTest(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())
}

func TestHostAnnounceWithPeers(t *testing.T) {
	assert := assert.New(t)
	host, _, _, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())
}

func TestHostPublishNoPeers(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))
}

func TestHostPublishPeers(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))
}

func TestHostReprovideNoPeers(t *testing.T) {
	assert := assert.New(t)
	host, ctrl := setUpHostTest(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Reprovide(context.Background(), true))
}

func TestHostReprovideWithPeers(t *testing.T) {
	assert := assert.New(t)
	host, _, _, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Reprovide(context.Background(), true))
}

func TestHostReprovideWithPeersAndBlocks(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	assert.Nil(host.Reprovide(context.Background(), true))
}

func TestHostReprovideWithPeersAndContent(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	assert.Nil(host.Reprovide(context.Background(), true))
}

func TestHostFindProvidersWithPeersAndContent(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	ch := host.FindProviders(context.Background(), blockMap[0])
	provider := <-ch
	assert.Equal(host.GetID(), provider.ID.Pretty())
}

func TestHostFindProvidersWithNoPeersAndContent(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	ch := host.FindProviders(context.Background(), blockMap[0])
	select {
	case <-ch:
		assert.FailNow("Did not expect peer info here")
	default:
		return
	}
}

func TestHostLockInvalidFile(t *testing.T) {
	assert := assert.New(t)
	host, _, _, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	token, err := host.Lock(context.Background(), privKey, "test", "test.txt")
	assert.NotNil(err)
	assert.Nil(token)
}

func TestHostLock(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	token, err := host.Lock(context.Background(), privKey, "test", "test.txt")
	assert.Nil(err)
	assert.NotEmpty(token.Token)
}

func TestHostUnlockInvalidFile(t *testing.T) {
	assert := assert.New(t)
	host, _, _, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	privKey, err := GenerateKeyPair()
	assert.Nil(err)

	err = host.Unlock(context.Background(), privKey, "test", "test.txt", nil)
	assert.NotNil(err)
}

func TestHostUnlock(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	token, err := host.Lock(context.Background(), privKey, "test", "test.txt")
	assert.Nil(err)
	assert.NotEmpty(token.Token)

	assert.Nil(host.Unlock(context.Background(), privKey, "test", "test.txt", token))
}

func TestHostUnlockInvalidToken(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	token, err := host.Lock(context.Background(), privKey, "test", "test.txt")
	assert.Nil(err)
	assert.NotEmpty(token.Token)

	token.Token = "abc"

	assert.NotNil(host.Unlock(context.Background(), privKey, "test", "test.txt", token))
}

func TestHostPublishAndLock(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	token, err := host.PublishAndLock(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData())))
	assert.Nil(err)
	assert.NotNil(token)
	assert.NotEmpty(token.Token)
}

func TestHostPublishAndLockUnlock(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	token, err := host.PublishAndLock(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData())))
	assert.Nil(err)
	assert.NotNil(token)
	assert.NotEmpty(token.Token)

	assert.Nil(host.Unlock(context.Background(), privKey, "test", "test.txt", token))
}

func TestHostGetNotFound(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	object, err := host.GetContent(context.Background(), privKey.GetPublic(), "test", "notfound.txt")
	assert.NotNil(err)
	assert.Nil(object)
}

func TestHostGetInvalidKey(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	privKey2, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey2.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	object, err := host.GetContent(context.Background(), privKey2.GetPublic(), "test", "test.txt")
	assert.NotNil(err)
	assert.Nil(object)
}

func TestHostGetNoPeers(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithDsAndBs(t)
	defer setDownHostTest(ctrl)

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	object, err := host.GetContent(context.Background(), privKey.GetPublic(), "test", "test.txt")
	assert.Nil(err)

	assert.Equal(int64(len(block.RawData())), object.Size)
	assert.Equal(1, len(object.Blocks))
	assert.False(object.Delete)
	assert.Nil(object.Lock)
	assert.Equal(uint64(0), object.CacheTTL)
}

func TestHostGetPeers(t *testing.T) {
	assert := assert.New(t)
	host, _, bs, ctrl := setUpHostTestWithRelay(t)
	defer setDownHostTest(ctrl)

	assert.Nil(host.Announce())

	block := blocks.NewBlock([]byte("abc"))
	assert.Nil(bs.Put(block))

	blockMap := interfaces.BlockMap{}
	blockMap[0] = &pb.BlockMetadata{
		Start: 0,
		Size:  int64(len(block.RawData())),
		Cid:   block.Cid().Bytes(),
	}

	privKey, err := GenerateKeyPair()
	assert.Nil(err)
	assert.Nil(host.PutPublicKey(privKey.GetPublic()))

	cipherKey := []byte("0123456789abcdef")

	assert.Nil(host.Publish(context.Background(), privKey, cipherKey, "test", "test.txt", blockMap, time.Now(), int64(len(block.RawData()))))

	host2Settings := &options.Settings{
		BootstrapPeers: host.(*hostImpl).settings.BootstrapPeers,
	}
	host2 := setUpHostBasic(t, host2Settings, nil, nil)
	assert.Nil(host2.Announce())

	object, err := host2.GetContent(context.Background(), privKey.GetPublic(), "test", "test.txt")
	assert.Nil(err)

	assert.Equal(int64(len(block.RawData())), object.Size)
	assert.Equal(1, len(object.Blocks))
	assert.False(object.Delete)
	assert.Nil(object.Lock)
	assert.Equal(uint64(0), object.CacheTTL)
}

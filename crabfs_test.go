package crabfs

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ipfsDatastore "github.com/ipfs/go-datastore"
	ipfsDatastoreSync "github.com/ipfs/go-datastore/sync"
	ipfsBlockstore "github.com/ipfs/go-ipfs-blockstore"
	crabfsCrypto "github.com/runletapp/crabfs/crypto"
	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/mocks"
	"github.com/runletapp/crabfs/options"
	pb "github.com/runletapp/crabfs/protos"
)

func setUpCrabFSTest(t *testing.T) (*crabFS, *gomock.Controller) {
	ctrl := setUpBasicTest(t)

	mockFetcher := func(ctx context.Context, fs interfaces.Core, object *pb.CrabObject, privateKey crabfsCrypto.PrivKey) (interfaces.Fetcher, error) {
		return mocks.NewMockFetcher(ctrl), nil
	}

	return setUpCrabFSTestWithFetcherFactory(t, ctrl, mockFetcher)
}

func setUpCrabFSTestWithFetcherFactory(t *testing.T, ctrl *gomock.Controller, mockFetcher interfaces.FetcherFactory) (*crabFS, *gomock.Controller) {
	host := mocks.NewMockHost(ctrl)
	settings := options.Settings{}
	settings.SetDefaults()

	datastore := ipfsDatastoreSync.MutexWrap(ipfsDatastore.NewMapDatastore())

	blockstore := ipfsBlockstore.NewBlockstore(datastore)

	fs := &crabFS{
		settings: &settings,
		host:     host,

		datastore:      datastore,
		blockstore:     blockstore,
		fetcherFactory: mockFetcher,
	}

	return fs, ctrl
}

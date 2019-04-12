package crabfs

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"log"
	"os"
	"path"
	"sync"
	"time"

	cid "github.com/ipfs/go-cid"
	"github.com/runletapp/crabfs/options"
	billy "gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"

	"github.com/patrickmn/go-cache"

	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
)

var (
	// ErrNotFound the request content id does not have providers
	ErrNotFound = errors.New("Not found")
	// ErrReadOnly the file is marked as read only
	ErrReadOnly = errors.New("Write on a read only file")
)

// CrabFS struct
type CrabFS struct {
	billy.Filesystem

	BucketName string

	mountFS billy.Filesystem

	privateKey *libp2pCrypto.RsaPrivateKey

	bootstrapPeers []string

	ctx       context.Context
	ctxCancel context.CancelFunc

	host *Host

	hashCache *cache.Cache

	openedFileStreams      map[string]*StreamContext
	openedFileStreamsMutex *sync.RWMutex
}

// New creates a new CrabFS instance
func New(mountPath string, opts ...options.Option) (*CrabFS, error) {
	mountFS := osfs.New(mountPath)

	var settings = &options.Settings{}
	if err := settings.SetDefaults(); err != nil {
		return nil, err
	}

	for _, option := range opts {
		if err := option(settings); err != nil {
			return nil, err
		}
	}

	var privateKey *libp2pCrypto.RsaPrivateKey
	var err error
	if settings.PrivateKey == nil {
		privKey, _, err := libp2pCrypto.GenerateRSAKeyPair(2048, rand.Reader)
		if err != nil {
			return nil, err
		}
		privateKey = privKey.(*libp2pCrypto.RsaPrivateKey)
	} else {
		privateKey, err = ReadPrivateKey(settings.PrivateKey)
		if err != nil {
			return nil, err
		}
	}

	childCtx, cancel := context.WithCancel(settings.Context)

	host, err := HostNew(childCtx, settings.Port, privateKey, mountFS)
	if err != nil {
		cancel()
		return nil, err
	}

	hashCache := cache.New(1*time.Hour, 1*time.Hour)

	fs := &CrabFS{
		mountFS: mountFS,

		BucketName: settings.BucketName,

		privateKey: privateKey,

		bootstrapPeers: settings.BootstrapPeers,

		ctx:       childCtx,
		ctxCancel: cancel,

		host: host,

		hashCache: hashCache,

		openedFileStreams:      make(map[string]*StreamContext),
		openedFileStreamsMutex: &sync.RWMutex{},
	}

	if !settings.RelayOnly {
		fs.background()
	}

	return fs, nil
}

// Close stops the node and closes all open connections
func (fs *CrabFS) Close() error {
	fs.ctxCancel()
	return nil
}

func (fs *CrabFS) background() {
	if err := fs.bootstrap(); err != nil {
		log.Printf("Bootstrap error: %v", err)
	}

	// Wait a bit after bootstraping
	<-time.After(500 * time.Millisecond)

	if err := fs.announce(); err != nil {
		log.Printf("Announce error: %v", err)
	}
}

// IsClosed check if the node has exited
func (fs *CrabFS) IsClosed() bool {
	return fs.ctx.Err() != nil
}

// GetHostID returns the id of this p2p host
func (fs *CrabFS) GetHostID() string {
	return fs.host.GetID()
}

func (fs *CrabFS) bootstrap() error {
	for _, addr := range fs.bootstrapPeers {
		if err := fs.host.ConnectToPeer(addr); err != nil {
			// log.Printf("Could not connect: %v", err)
		}
	}

	return nil
}

// PublishLocalFiles announce the local files to the network
func (fs *CrabFS) PublishLocalFiles(ctx context.Context, dir string, verify bool) error {
	files, err := fs.mountFS.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if file.IsDir() {
			err = fs.PublishLocalFiles(ctx, fs.mountFS.Join(dir, file.Name()), verify)
			if err != nil {
				return err
			}

		} else {
			_, err = fs.PublishFile(ctx, fs.mountFS.Join(dir, file.Name()), verify)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PublishFile publishes 'filename' and returns the cid related to it
func (fs *CrabFS) PublishFile(ctx context.Context, filename string, verify bool) (*cid.Cid, error) {
	file, err := fs.mountFS.Open(filename)
	if err != nil {
		return nil, err
	}

	stat, err := fs.mountFS.Stat(filename)
	if err != nil {
		return nil, err
	}

	crabfile := localFileNew(file, stat, fs.hashCache, os.O_RDONLY)
	defer crabfile.Close()

	cid, err := crabfile.CalcCID()
	if err != nil {
		return nil, err
	}

	if verify {
		upstreamRecord, err := fs.GetContentRecord(ctx, filename)
		if err != nil && err != ErrNotFound {
			return nil, err
		} else if err == nil {
			mtime := stat.ModTime()
			if fs.comprareWithUpstream(upstreamRecord, cid, &mtime) < 0 {
				// File contents difer, since we are not updating, restore the file to the upstream version
				if err := fs.mountFS.Remove(filename); err != nil {
					return nil, err
				}
				return upstreamRecord.ContentID, nil
			}
		}
	}

	return cid, fs.publishContentID(filename, stat, cid)
}

// GetAddrs get the addresses that this node is bound to
func (fs *CrabFS) GetAddrs() []string {
	return fs.host.GetAddrs()
}

func (fs *CrabFS) announce() error {
	err := fs.host.Announce()
	if err != nil {
		return err
	}

	return fs.PublishLocalFiles(fs.ctx, ".", true)
}

// publishContentID announce that we can provide the content id 'contentID'
func (fs *CrabFS) publishContentID(pathName string, fileInfo os.FileInfo, contentID *cid.Cid) error {
	var record *DHTNameRecord

	if fileInfo != nil && contentID != nil {
		record = &DHTNameRecord{
			DHTNameRecordV1: DHTNameRecordV1{
				Perm:   uint32(fileInfo.Mode().Perm()),
				Length: fileInfo.Size(),
				Mtime:  fileInfo.ModTime().UTC().Format(time.RFC3339Nano),
			},
		}
	} else {
		record = &DHTNameRecord{}
	}

	return fs.host.AnnounceContent(pathName, contentID, record)
}

// GetContentRecord resolves the pathName to a content id
func (fs *CrabFS) GetContentRecord(ctx context.Context, pathName string) (*DHTNameRecord, error) {
	return fs.host.GetContentRecord(ctx, pathName)
}

// GetProviders resolves the content id to a list of peers that provides that content
func (fs *CrabFS) GetProviders(ctx context.Context, contentID *cid.Cid) ([][]string, error) {
	return fs.host.GetProviders(ctx, contentID)
}

func (fs *CrabFS) changeMtime(filename string, mtime time.Time) error {
	absFilename := path.Join(fs.mountFS.Root(), fs.mountFS.Join(filename))
	return os.Chtimes(absFilename, mtime, mtime)
}

func (fs *CrabFS) changePerm(filename string, perm uint32) error {
	absFilename := path.Join(fs.mountFS.Root(), fs.mountFS.Join(filename))
	return os.Chmod(absFilename, os.FileMode(perm))
}

func (fs *CrabFS) comprareWithUpstream(record *DHTNameRecord, localCid *cid.Cid, localMTime *time.Time) int {
	upstreamCid := record.ContentID
	if !upstreamCid.Equals(*localCid) {
		mtime := localMTime.UTC().Format(time.RFC3339Nano)
		if bytes.Compare([]byte(mtime), []byte(record.Mtime)) < 0 {
			return -1
		}
	}

	return 1
}

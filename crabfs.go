package crabfs

import (
	"context"
	"crypto/rand"
	"log"
	"os"
	"path"
	"sync"
	"time"

	cid "github.com/ipfs/go-cid"
	"github.com/runletapp/crabfs/options"
	billy "gopkg.in/src-d/go-billy.v4"

	"github.com/patrickmn/go-cache"

	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	libp2pRouting "github.com/libp2p/go-libp2p-routing"
)

var (
	// ErrNotFound the request content id does not have providers
	ErrNotFound = libp2pRouting.ErrNotFound
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
func New(mountFS billy.Filesystem, opts ...options.Option) (*CrabFS, error) {
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

	host, err := HostNew(settings.Context, settings.Port, privateKey, mountFS)
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
		go fs.background()
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

	<-fs.ctx.Done()
	fs.Close()
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
			log.Printf("Could not connect: %v", err)
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

	fileBlock, err := FileNew(filename, file, fs.hashCache)
	if err != nil {
		file.Close()
		return nil, err
	}
	defer fileBlock.Close()

	stat, err := fs.mountFS.Stat(filename)
	if err != nil {
		return nil, err
	}
	mtime := stat.ModTime().UTC()

	_, err = fileBlock.CalcHash(&mtime)
	if err != nil {
		return nil, err
	}

	cid := fileBlock.GetCID()

	if verify {
		upstreamRecord, err := fs.GetContentRecord(ctx, filename)
		if err != nil && err != ErrNotFound {
			return nil, err
		} else if err == nil {
			upstreamCid := upstreamRecord.ContentID
			if !upstreamCid.Equals(cid) {
				// File contents difer, since we are not updating, restore the file to the upstream version
				if err := fs.mountFS.Remove(filename); err != nil {
					return nil, err
				}
				return upstreamCid, nil
			}
		}
	}

	return &cid, fs.announceContentID(filename, &cid)
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

	return fs.announceContent(fs.ctx)
}

func (fs *CrabFS) announceContent(ctx context.Context) error {
	err := fs.PublishLocalFiles(ctx, ".", true)
	if err != nil {
		return err
	}

	return nil
}

// announceContentID announce that we can provide the content id 'contentID'
func (fs *CrabFS) announceContentID(pathName string, contentID *cid.Cid) error {
	stat, err := fs.mountFS.Stat(pathName)
	if err != nil {
		return err
	}

	record := &DHTNameRecord{
		DHTNameRecordV1: DHTNameRecordV1{
			Perm:   uint32(stat.Mode().Perm()),
			Length: stat.Size(),
			Mtime:  stat.ModTime().UTC().Format(time.RFC3339),
		},
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

// CreateFileStream create a reader to get data from the network
func (fs *CrabFS) CreateFileStream(ctx context.Context, pathName string) (billy.File, error) {
	fs.openedFileStreamsMutex.RLock()
	defer fs.openedFileStreamsMutex.RUnlock()

	if _, prs := fs.openedFileStreams[pathName]; prs {
		return fs.mountFS.Open(pathName)
	}

	upstreamRecord, err := fs.GetContentRecord(ctx, pathName)
	if err != nil {
		return nil, err
	}

	if err := fs.mountFS.MkdirAll(path.Dir(pathName), 0755); err != nil {
		return nil, err
	}

	tmpFile, err := fs.mountFS.Create(pathName)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			tmpFile.Close()

			mtime, err := time.Parse(time.RFC3339, upstreamRecord.Mtime)
			if err == nil {
				fs.changeMtime(pathName, mtime)
			}

			fs.changePerm(pathName, upstreamRecord.Perm)
		}()

		streamCtx, streamCtxCancel := context.WithCancel(ctx)
		defer streamCtxCancel()

		fs.openedFileStreamsMutex.Lock()
		fs.openedFileStreams[pathName] = &StreamContext{
			Ctx:    streamCtx,
			Cancel: streamCtxCancel,
		}
		fs.openedFileStreamsMutex.Unlock()

		streamCh, err := fs.host.CreateStream(streamCtx, upstreamRecord.ContentID, pathName)
		if err != nil {
			return
		}

		BUFFERSIZE := 500 * 1024
		data := make([]byte, BUFFERSIZE)

		var totalRead int64

	channelLoop:
		for {
			select {
			case stream, ok := <-streamCh:
				if !ok {
					return
				}

				var totalChannelRead int64

				for ctx.Err() == nil && totalChannelRead < stream.Length {
					n, err := stream.Read(data)
					if err != nil {
						return
					}

					n, err = tmpFile.Write(data[:n])
					if err != nil {
						return
					}

					totalChannelRead += int64(n)
					totalRead += int64(n)
					if totalRead >= upstreamRecord.Length {
						break channelLoop
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// We open a new file so we don't mess up file descriptors
	return fs.mountFS.Open(tmpFile.Name())
}

func (fs *CrabFS) changeMtime(filename string, mtime time.Time) error {
	return os.Chtimes(filename, mtime, mtime)
}

func (fs *CrabFS) changePerm(filename string, perm uint32) error {
	return os.Chmod(filename, os.FileMode(perm))
}

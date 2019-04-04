package crabfs

import (
	"context"
	"log"

	cid "github.com/ipfs/go-cid"
	billy "gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

// CrabFS struct
type CrabFS struct {
	billy.Filesystem

	tmpfs billy.Filesystem

	BucketName string

	mountLocation string
	mountFS       billy.Filesystem

	bootstrapPeers []string

	ctx       context.Context
	ctxCancel context.CancelFunc

	host         *Host
	discoveryKey string
}

// New creates a new CrabFS instance
func New(bucketName string, mountLocation string, discoveryKey string, port int, tmpfs billy.Filesystem) (*CrabFS, error) {
	return NewWithContext(context.Background(), bucketName, mountLocation, discoveryKey, port, tmpfs)
}

// NewWithContext creates a new CrabFS instance with a context
func NewWithContext(ctx context.Context, bucketName string, mountLocation string, discoveryKey string, port int, tmpfs billy.Filesystem) (*CrabFS, error) {
	childCtx, cancel := context.WithCancel(ctx)

	host, err := HostNew(ctx, port, discoveryKey)
	if err != nil {
		cancel()
		return nil, err
	}

	mountFS := osfs.New(mountLocation)

	fs := &CrabFS{
		tmpfs: tmpfs,

		BucketName: bucketName,

		mountLocation: mountLocation,
		mountFS:       mountFS,

		bootstrapPeers: []string{},

		ctx:       childCtx,
		ctxCancel: cancel,

		host:         host,
		discoveryKey: discoveryKey,
	}

	go fs.background()

	return fs, nil
}

// Close stops the node and closes all open connections
func (fs *CrabFS) Close() error {
	fs.ctxCancel()
	return nil
}

func (fs *CrabFS) background() {
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

// Bootstrap batch connect to each peer in peers
func (fs *CrabFS) Bootstrap(peers []string) error {
	for _, addr := range peers {
		if err := fs.host.ConnectToPeer(addr); err != nil {
			return err
		}
	}

	return nil
}

// PublishLocalFiles announce the local files to the network
func (fs *CrabFS) PublishLocalFiles(dir string) (*cid.Cid, error) {
	files, err := fs.mountFS.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	dirBlock, err := DirNew(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		var newCid *cid.Cid
		if file.IsDir() {
			newCid, err = fs.PublishLocalFiles(fs.mountFS.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}

		} else {
			newCid, err = fs.PublishFile(fs.mountFS.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
		}

		dirBlock.Children = append(dirBlock.Children, newCid)
	}

	_, err = dirBlock.CalcHash()
	if err != nil {
		return nil, err
	}

	cid := dirBlock.GetCID()

	return &cid, fs.announceContentID(dir, &cid)
}

// PublishFile publishes 'filename' and returns the cid related to it
func (fs *CrabFS) PublishFile(filename string) (*cid.Cid, error) {
	file, err := fs.mountFS.Open(filename)
	if err != nil {
		return nil, err
	}

	fileBlock, err := FileNew(filename, file)
	if err != nil {
		return nil, err
	}

	_, err = fileBlock.CalcHash()
	if err != nil {
		return nil, err
	}

	cid := fileBlock.GetCID()

	return &cid, fs.announceContentID(filename, &cid)
}

// GetAddrs get the addresses that this node is bound to
func (fs *CrabFS) GetAddrs() []string {
	return fs.host.GetAddrs()
}

// Announce announce this host in the network
func (fs *CrabFS) Announce() error {
	err := fs.host.Announce()
	if err != nil {
		return err
	}

	return fs.AnnounceContent()
}

// AnnounceContent update the content root resolver
func (fs *CrabFS) AnnounceContent() error {
	baseCID, err := fs.PublishLocalFiles(".")
	if err != nil {
		return err
	}

	return fs.announceContentID("/", baseCID)
}

// announceContentID announce that we can provide the content id 'contentID'
func (fs *CrabFS) announceContentID(pathName string, contentID *cid.Cid) error {
	log.Printf("Add new cid for '%s': %v", pathName, contentID.String())

	return fs.host.AnnounceContent(pathName, contentID)
}

// GetContentID resolves the pathName to a content id
func (fs *CrabFS) GetContentID(ctx context.Context, pathName string) (*cid.Cid, error) {
	return fs.host.GetContentID(ctx, pathName)
}

// GetProviders resolves the content id to a list of peers that provides that content
func (fs *CrabFS) GetProviders(ctx context.Context, contentID *cid.Cid) ([][]string, error) {
	return fs.host.GetProviders(ctx, contentID)
}

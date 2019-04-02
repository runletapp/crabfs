package crabfs

import "context"

// CrabFS struct
type CrabFS struct {
	MountPoint string

	bootstrapPeers []string

	ctx       context.Context
	ctxCancel context.CancelFunc

	host         *Host
	discoveryKey string
}

// New creates a new CrabFS instance
func New(mountPoint string, discoveryKey string) (*CrabFS, error) {
	return NewWithContext(context.Background(), mountPoint, discoveryKey)
}

// NewWithContext creates a new CrabFS instance with a context
func NewWithContext(ctx context.Context, mountPoint string, discoveryKey string) (*CrabFS, error) {
	childCtx, cancel := context.WithCancel(ctx)

	host, err := HostNew(ctx, 2525, discoveryKey)
	if err != nil {
		cancel()
		return nil, err
	}

	fs := &CrabFS{
		MountPoint: mountPoint,

		bootstrapPeers: []string{},

		ctx:       childCtx,
		ctxCancel: cancel,

		host:         host,
		discoveryKey: discoveryKey,
	}

	go fs.waitForContext()

	return fs, nil
}

// Close stops the node and closes all open connections
func (fs *CrabFS) Close() error {
	fs.ctxCancel()
	return nil
}

func (fs *CrabFS) waitForContext() {
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

// GetAddrs get the addresses that this node is bound to
func (fs *CrabFS) GetAddrs() []string {
	return fs.host.GetAddrs()
}

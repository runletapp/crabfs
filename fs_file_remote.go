package crabfs

import (
	"context"
	fmt "fmt"

	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	billy "gopkg.in/src-d/go-billy.v4"
)

var _ File = remoteFile{}

type remoteFile struct {
	Object

	ctx    context.Context
	name   string
	cid    *cid.Cid
	record *DHTNameRecord
	host   *Host

	targetFile billy.File

	OnClose OnCloseFunc

	pullerContext       context.Context
	pullerContextCancel context.CancelFunc
}

func remoteFileNew(ctx context.Context, name string, record *DHTNameRecord, host *Host, targetFile billy.File) (*remoteFile, error) {
	pCtx, pCancel := context.WithCancel(ctx)

	file := &remoteFile{
		ctx:    ctx,
		name:   name,
		cid:    record.ContentID,
		record: record,
		host:   host,

		targetFile: targetFile,

		pullerContext:       pCtx,
		pullerContextCancel: pCancel,
	}

	go file.readLoop()

	return file, nil
}

func (file remoteFile) PullerContext() context.Context {
	return file.pullerContext
}

func (file remoteFile) Read(b []byte) (int, error) {
	return file.targetFile.Read(b)
}

func (file remoteFile) ReadAt(b []byte, off int64) (int, error) {
	return file.targetFile.ReadAt(b, off)
}

func (file remoteFile) Seek(off int64, whence int) (int64, error) {
	return file.targetFile.Seek(off, whence)
}

func (file remoteFile) Write(b []byte) (int, error) {
	return file.targetFile.Write(b)
}

func (file remoteFile) Name() string {
	return file.name
}

func (file remoteFile) GetType() ObjectType {
	return ObjectType_File
}

func (file remoteFile) CalcHash() (multihash.Multihash, error) {
	return nil, nil
}

func (file remoteFile) CalcCID() (*cid.Cid, error) {
	return file.cid, nil
}

func (file remoteFile) GetSize() int64 {
	return file.record.Length
}

func (file remoteFile) Close() error {
	if file.OnClose != nil {
		if err := file.OnClose(file); err != nil {
			return err
		}
	}

	return nil
}

// Lock locks the file like e.g. flock. It protects against access from
// other processes.
// Note: Not supported
func (file remoteFile) Lock() error {
	return fmt.Errorf("Lock: Not supported")
}

// Unlock unlocks the file.
// Note: Not supported
func (file remoteFile) Unlock() error {
	return fmt.Errorf("Unlock: Not supported")
}

// Truncate the file.
// Note: Not supported
func (file remoteFile) Truncate(size int64) error {
	return fmt.Errorf("Truncate: Not supported")
}

func (file remoteFile) readLoop() {
	streamCtx, streamCtxCancel := context.WithCancel(file.ctx)
	defer streamCtxCancel()
	defer file.pullerContextCancel()

	streamCh, err := file.host.CreateStream(streamCtx, file.record.ContentID, file.Name())
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

			for file.ctx.Err() == nil && totalChannelRead < stream.Length {
				n, err := stream.Read(data)
				if err != nil {
					return
				}

				n, err = file.targetFile.Write(data[:n])
				if err != nil {
					return
				}

				totalChannelRead += int64(n)
				totalRead += int64(n)
				if totalRead >= file.record.Length {
					break channelLoop
				}
			}
		case <-file.ctx.Done():
			return
		}
	}
}

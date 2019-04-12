package crabfs

import (
	"bytes"
	"crypto/sha256"
	fmt "fmt"
	"io"
	"os"
	"time"

	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	"github.com/patrickmn/go-cache"
	billy "gopkg.in/src-d/go-billy.v4"
)

var _ File = localFile{}

type localFile struct {
	billy.File
	Object

	OnClose OnCloseFunc

	fileInfo  os.FileInfo
	hashCache *cache.Cache

	mode int
}

func localFileNew(underlyingFile billy.File, fileInfo os.FileInfo, hashCache *cache.Cache, mode int) *localFile {
	return &localFile{
		File:      underlyingFile,
		fileInfo:  fileInfo,
		hashCache: hashCache,
		mode:      mode,
	}
}

func (file *localFile) getCacheKey(mtime *time.Time) string {
	var mtimeKey string
	if mtime != nil {
		mtimeKey = mtime.UTC().Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("file/local/%s/%s", file.Name(), mtimeKey)
}

func (file localFile) GetType() ObjectType {
	return ObjectType_File
}

func (file localFile) Write(b []byte) (int, error) {
	if file.mode == os.O_RDONLY {
		return 0, ErrReadOnly
	}

	return file.File.Write(b)
}

func (file localFile) Close() error {
	if file.OnClose != nil {
		if err := file.OnClose(file); err != nil {
			return err
		}
	}

	return file.File.Close()
}

func (file localFile) CalcHash() (multihash.Multihash, error) {
	mtime := file.fileInfo.ModTime()

	cacheKey := file.getCacheKey(&mtime)

	var hash multihash.Multihash

	hashI, found := file.hashCache.Get(cacheKey)
	if found {
		hash = hashI.(multihash.Multihash)
		return hash, nil
	}

	h := sha256.New()
	pos := int64(0)
	size := file.fileInfo.Size()
	buffer := make([]byte, 500)

	for pos < size {
		n, err := file.ReadAt(buffer, pos)
		if err != nil && err != io.EOF {
			return nil, err
		}
		_, err = io.Copy(h, bytes.NewReader(buffer[:n]))
		if err != nil {
			return nil, err
		}

		pos += int64(n)
	}
	hash, err := multihash.Encode(h.Sum(nil), multihash.SHA3_256)
	if err != nil {
		return nil, err
	}

	file.hashCache.SetDefault(cacheKey, hash)

	return hash, nil
}

func (file localFile) CalcCID() (*cid.Cid, error) {
	hash, err := file.CalcHash()
	if err != nil {
		return nil, err
	}

	c := cid.NewCidV1(cid.Raw, hash)

	return &c, nil
}

func (file localFile) GetSize() int64 {
	return file.fileInfo.Size()
}

// Lock locks the file like e.g. flock. It protects against access from
// other processes.
// Note: Not supported
func (file localFile) Lock() error {
	return fmt.Errorf("Lock: Not supported")
}

// Unlock unlocks the file.
// Note: Not supported
func (file localFile) Unlock() error {
	return fmt.Errorf("Unlock: Not supported")
}

// Truncate the file.
// Note: Not supported
func (file localFile) Truncate(size int64) error {
	return fmt.Errorf("Truncate: Not supported")
}

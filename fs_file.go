package crabfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	multihash "github.com/multiformats/go-multihash"
	"github.com/patrickmn/go-cache"
	"gopkg.in/src-d/go-billy.v4"
)

// File p2p file abstraction
type File struct {
	Block

	billy.File

	Size int64

	OnFileClose func(file *File) error

	hashCache *cache.Cache
}

// FileNew creates a new file wrapper
func FileNew(filename string, underlyingFile billy.File, hashCache *cache.Cache) (*File, error) {
	// Save the current position, so we can restore later
	pos, err := underlyingFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	size, err := underlyingFile.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	_, err = underlyingFile.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return &File{
		Block: Block{
			name: filename,
			Type: BlockTypeFILE,
		},
		File:      underlyingFile,
		Size:      size,
		hashCache: hashCache,
	}, nil
}

func (file *File) getCacheKey(mtime *time.Time) string {
	var mtimeKey string
	if mtime != nil {
		mtimeKey = mtime.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("file-%s-%s", file.Name(), mtimeKey)
}

// CalcHash return the current hash of the block
func (file *File) CalcHash(mtime *time.Time) (multihash.Multihash, error) {
	cacheKey := file.getCacheKey(mtime)

	var hash multihash.Multihash

	hashI, found := file.hashCache.Get(cacheKey)
	if found {
		hash = hashI.(multihash.Multihash)
		file.mhash = hash
		return hash, nil
	}

	buffer, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	hash, err = multihash.Sum(buffer, multihash.SHA3_256, -1)
	if err != nil {
		return nil, err
	}

	file.mhash = hash

	file.hashCache.SetDefault(cacheKey, hash)

	return hash, nil
}

// Name returns the name of the file as presented to Open.
func (file *File) Name() string {
	return file.name
}

func (file *File) Write(p []byte) (int, error) {
	return file.File.Write(p)
}

// Close close this file and submit changes
func (file *File) Close() error {
	if file.OnFileClose != nil {
		err := file.OnFileClose(file)
		return err
	}

	return file.File.Close()
}

func (file *File) Read(p []byte) (int, error) {
	return file.File.Read(p)
}

// ReadAt reads len(p) bytes into p starting at offset off in the underlying input source. It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
// When ReadAt returns n < len(p), it returns a non-nil error explaining why more bytes were not returned. In this respect, ReadAt is stricter than Read.
// Even if ReadAt returns n < len(p), it may use all of p as scratch space during the call. If some data is available but not len(p) bytes, ReadAt blocks until either all the data is available or an error occurs. In this respect ReadAt is different from Read.
// If the n = len(p) bytes returned by ReadAt are at the end of the input source, ReadAt may return either err == EOF or err == nil.
func (file *File) ReadAt(p []byte, off int64) (int, error) {
	return file.File.ReadAt(p, off)
}

// Seek sets the offset for the next Read or Write to offset, interpreted according to whence: SeekStart means relative to the start of the file, SeekCurrent means relative to the current offset, and SeekEnd means relative to the end. Seek returns the new offset relative to the start of the file and an error, if any.
// Seeking to an offset before the start of the file is an error. Seeking to any positive offset is legal, but the behavior of subsequent I/O operations on the underlying object is implementation-dependent.
func (file *File) Seek(offset int64, whence int) (int64, error) {
	return file.File.Seek(offset, whence)
}

// Lock locks the file like e.g. flock. It protects against access from
// other processes.
// Note: Not supported
func (file *File) Lock() error {
	return fmt.Errorf("Lock: Not supported")
}

// Unlock unlocks the file.
// Note: Not supported
func (file *File) Unlock() error {
	return fmt.Errorf("Unlock: Not supported")
}

// Truncate the file.
// Note: Not supported
func (file *File) Truncate(size int64) error {
	return fmt.Errorf("Truncate: Not supported")
}

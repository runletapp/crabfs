package crabfs

import (
	"context"
	"os"
	"time"

	billy "gopkg.in/src-d/go-billy.v4"
)

// OnFileClose handler to process tmpfs file after closing
func (fs *CrabFS) OnFileClose(file *File) error {
	_, err := fs.PublishFile(context.Background(), file.Name(), false)
	return err
}

// Create creates the named file with mode 0666 (before umask), truncating
// it if it already exists. If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
func (fs *CrabFS) Create(filename string) (billy.File, error) {
	file, err := fs.mountFS.Create(filename)
	if err != nil {
		return nil, err
	}

	crabfile, err := FileNew(filename, file, fs.hashCache)
	if err != nil {
		return nil, err
	}

	crabfile.OnFileClose = fs.OnFileClose

	return crabfile, nil
}

// Open opens the named file for reading. If successful, methods on the
// returned file can be used for reading; the associated file descriptor has
// mode O_RDONLY.
func (fs *CrabFS) Open(filename string) (billy.File, error) {
	return fs.OpenContext(context.Background(), filename)
}

// OpenContext opens the named file for reading with the specified context. If successful, methods on the
// returned file can be used for reading; the associated file descriptor has
// mode O_RDONLY.
func (fs *CrabFS) OpenContext(ctx context.Context, filename string) (billy.File, error) {
	upstreamRecord, err := fs.GetContentRecord(ctx, filename)
	if err != nil {
		return nil, err
	}

	underlyingFile, err := fs.mountFS.Open(filename)
	if err != nil {
		return fs.openFileStream(ctx, filename)
	}

	// the file exists locally, validate its contents

	fileBlock, err := FileNew(filename, underlyingFile, fs.hashCache)
	if err != nil {
		return nil, err
	}

	stat, err := fs.mountFS.Stat(filename)
	if err != nil {
		return nil, err
	}
	mtime := stat.ModTime()

	if _, err := fileBlock.CalcHash(&mtime); err != nil {
		return nil, err
	}

	contentIDCalc := fileBlock.GetCID()

	if !contentIDCalc.Equals(*upstreamRecord.ContentID) {
		fileBlock.Close()
		return fs.openFileStream(ctx, filename)
	}

	return fileBlock, nil
}

func (fs *CrabFS) openFileStream(ctx context.Context, filename string) (billy.File, error) {
	underlyingFile, err := fs.CreateFileStream(ctx, filename)
	if err != nil {
		return nil, err
	}

	fileBlock, err := FileNew(filename, underlyingFile, fs.hashCache)
	if err != nil {
		return nil, err
	}
	fileBlock.OnFileClose = fs.OnFileClose

	return fileBlock, nil
}

// OpenFile is the generalized open call; most users will use Open or Create
// instead. It opens the named file with specified flag (O_RDONLY etc.) and
// perm, (0666 etc.) if applicable. If successful, methods on the returned
// File can be used for I/O.
// Note: not supported
func (fs *CrabFS) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	return nil, billy.ErrNotSupported
}

// Stat returns a FileInfo describing the named file.
func (fs *CrabFS) Stat(filename string) (os.FileInfo, error) {
	return fs.StatContext(context.Background(), filename)
}

// StatContext returns a FileInfo describing the named file with context.
func (fs *CrabFS) StatContext(ctx context.Context, filename string) (os.FileInfo, error) {
	upstreamRecord, err := fs.GetContentRecord(ctx, filename)
	if err != nil {
		return nil, err
	}

	mtime, err := time.Parse(time.RFC3339, upstreamRecord.Mtime)
	if err != nil {
		mtime = time.Now().UTC()
	}

	fileInfo := FileInfo{
		name:  filename,
		size:  upstreamRecord.Length,
		mode:  upstreamRecord.Perm,
		mtime: mtime,
	}

	return &fileInfo, nil
}

// Rename renames (moves) oldpath to newpath. If newpath already exists and
// is not a directory, Rename replaces it. OS-specific restrictions may
// apply when oldpath and newpath are in different directories.
func (fs *CrabFS) Rename(oldpath, newpath string) error {
	return fs.RenameContext(context.Background(), oldpath, newpath)
}

// RenameContext renames (moves) oldpath to newpath with context. If newpath already exists and
// is not a directory, Rename replaces it. OS-specific restrictions may
// apply when oldpath and newpath are in different directories.
func (fs *CrabFS) RenameContext(ctx context.Context, oldpath, newpath string) error {
	upstreamRecord, err := fs.GetContentRecord(ctx, oldpath)
	if err != nil {
		return err
	}

	if err := fs.mountFS.Rename(oldpath, newpath); err != nil {
		return err
	}

	// Remove old path from the routing table
	if err := fs.announceContentID(oldpath, nil); err != nil {
		return err
	}

	return fs.announceContentID(newpath, upstreamRecord.ContentID)
}

// Remove removes the named file or directory.
func (fs *CrabFS) Remove(filename string) error {
	if err := fs.mountFS.Remove(filename); err != nil {
		return err
	}

	// Remove path from the routing table
	return fs.announceContentID(filename, nil)
}

// Join joins any number of path elements into a single path, adding a
// Separator if necessary. Join calls filepath.Clean on the result; in
// particular, all empty strings are ignored. On Windows, the result is a
// UNC path if and only if the first path element is a UNC path.
func (fs *CrabFS) Join(elem ...string) string {
	return fs.mountFS.Join(elem...)
}

// TempFile creates a new temporary file in the directory dir with a name
// beginning with prefix, opens the file for reading and writing, and
// returns the resulting *os.File. If dir is the empty string, TempFile
// uses the default directory for temporary files (see os.TempDir).
// Multiple programs calling TempFile simultaneously will not choose the
// same file. The caller can use f.Name() to find the pathname of the file.
// It is the caller's responsibility to remove the file when no longer
// needed.
func (fs *CrabFS) TempFile(dir, prefix string) (billy.File, error) {
	return fs.mountFS.TempFile(dir, prefix)
}

// ReadDir reads the directory named by dirname and returns a list of
// directory entries sorted by filename.
func (fs *CrabFS) ReadDir(path string) ([]os.FileInfo, error) {
	return fs.mountFS.ReadDir(path)
}

// MkdirAll creates a directory named path, along with any necessary
// parents, and returns nil, or else returns an error. The permission bits
// perm are used for all directories that MkdirAll creates. If path is/
// already a directory, MkdirAll does nothing and returns nil.
func (fs *CrabFS) MkdirAll(filename string, perm os.FileMode) error {
	return fs.mountFS.MkdirAll(filename, perm)
}

// Chroot returns a new filesystem from the same type where the new root is
// the given path. Files outside of the designated directory tree cannot be
// accessed.
func (fs *CrabFS) Chroot(path string) (billy.Filesystem, error) {
	return nil, billy.ErrNotSupported
}

// Root returns the root path of the filesystem.
func (fs *CrabFS) Root() string {
	return fs.mountFS.Root()
}

package crabfs

import (
	"os"

	billy "gopkg.in/src-d/go-billy.v4"
)

// Create creates the named file with mode 0666 (before umask), truncating
// it if it already exists. If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
func (fs *CrabFS) Create(filename string) (billy.File, error) {
	file, err := fs.tmpfs.Create(filename)
	if err != nil {
		return nil, err
	}

	crabfile, err := FileNew(filename, file)
	if err != nil {
		return nil, err
	}

	return crabfile, nil
}

// Open opens the named file for reading. If successful, methods on the
// returned file can be used for reading; the associated file descriptor has
// mode O_RDONLY.
func (fs *CrabFS) Open(filename string) (billy.File, error) {
	return fs.tmpfs.Open(filename)
}

// OpenFile is the generalized open call; most users will use Open or Create
// instead. It opens the named file with specified flag (O_RDONLY etc.) and
// perm, (0666 etc.) if applicable. If successful, methods on the returned
// File can be used for I/O.
func (fs *CrabFS) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	return fs.tmpfs.OpenFile(filename, flag, perm)
}

// Stat returns a FileInfo describing the named file.
func (fs *CrabFS) Stat(filename string) (os.FileInfo, error) {
	return fs.tmpfs.Stat(filename)
}

// Rename renames (moves) oldpath to newpath. If newpath already exists and
// is not a directory, Rename replaces it. OS-specific restrictions may
// apply when oldpath and newpath are in different directories.
func (fs *CrabFS) Rename(oldpath, newpath string) error {
	return fs.tmpfs.Rename(oldpath, newpath)
}

// Remove removes the named file or directory.
func (fs *CrabFS) Remove(filename string) error {
	return fs.tmpfs.Remove(filename)
}

// Join joins any number of path elements into a single path, adding a
// Separator if necessary. Join calls filepath.Clean on the result; in
// particular, all empty strings are ignored. On Windows, the result is a
// UNC path if and only if the first path element is a UNC path.
func (fs *CrabFS) Join(elem ...string) string {
	return fs.tmpfs.Join(elem...)
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
	return fs.tmpfs.TempFile(dir, prefix)
}

// ReadDir reads the directory named by dirname and returns a list of
// directory entries sorted by filename.
func (fs *CrabFS) ReadDir(path string) ([]os.FileInfo, error) {
	return fs.tmpfs.ReadDir(path)
}

// MkdirAll creates a directory named path, along with any necessary
// parents, and returns nil, or else returns an error. The permission bits
// perm are used for all directories that MkdirAll creates. If path is/
// already a directory, MkdirAll does nothing and returns nil.
func (fs *CrabFS) MkdirAll(filename string, perm os.FileMode) error {
	return fs.tmpfs.MkdirAll(filename, perm)
}

// Chroot returns a new filesystem from the same type where the new root is
// the given path. Files outside of the designated directory tree cannot be
// accessed.
func (fs *CrabFS) Chroot(path string) (billy.Filesystem, error) {
	return fs.tmpfs.Chroot(path)
}

// Root returns the root path of the filesystem.
func (fs *CrabFS) Root() string {
	return fs.BucketName
}

package interfaces

import (
	"context"
	"io"
	"time"

	pb "github.com/runletapp/crabfs/protos"
)

// Bucket bucket managemente interface
type Bucket interface {
	// Get opens a file stream
	Get(ctx context.Context, filename string) (Fetcher, error)

	// Put writes a file to the storage
	Put(ctx context.Context, filename string, file io.Reader, mtime time.Time) error

	// PutAndLock writes a file to the storage and locks it
	PutAndLock(ctx context.Context, filename string, file io.Reader, mtime time.Time) (*pb.LockToken, error)

	// Remove deletes a file from the storage
	Remove(ctx context.Context, filename string) error

	// Lock locks a file to avoid replublishing and overwritting during sequential updates from a single writer
	Lock(ctx context.Context, filename string) (*pb.LockToken, error)

	// IsLocked check if a file is locked
	IsLocked(ctx context.Context, filename string) (bool, error)

	// Unlock unlocks a file. See Lock
	Unlock(ctx context.Context, filename string, token *pb.LockToken) error

	Chroot(dir string) Bucket
}

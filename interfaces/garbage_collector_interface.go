package interfaces

import (
	"sync"
)

// GarbageCollector periodically runs a clean up in the blockstore to remove unused blocks
type GarbageCollector interface {
	// Start starts this garbage collector
	Start() error

	// Schedule schedules a clean up as soon as possible
	Schedule() error

	// Collect perform a clean up now
	Collect() error

	// Locker returns the garbage collector locker
	Locker() sync.Locker
}

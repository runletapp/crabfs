package crabfs

import (
	"gopkg.in/src-d/go-billy.v4"
)

// File p2p file abstraction
type File interface {
	billy.File
	Object
}

// OnCloseFunc handler for closing local files
type OnCloseFunc func(file File) error

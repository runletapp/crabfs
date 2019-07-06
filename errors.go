package crabfs

import "errors"

var (
	// ErrInvalidPrivateKey an invalid private key was supplied
	ErrInvalidPrivateKey = errors.New("Invalid private key")

	// ErrInvalidRoot an invalid path was given
	ErrInvalidRoot = errors.New("Invalid root path")

	// ErrInvalidBlockSize an invalid block size was given
	ErrInvalidBlockSize = errors.New("Invalid block size. Block size must be bigger than 0")

	// ErrInvalidOffset an invalid path was given
	ErrInvalidOffset = errors.New("Invalid offset")

	// ErrBlockNotFound the requested block was not found
	ErrBlockNotFound = errors.New("Block not found")

	// ErrObjectNotFound the requested object was not found
	ErrObjectNotFound = errors.New("Object not found")

	// ErrFileLocked the requested file is locked
	ErrFileLocked = errors.New("Request file is locked")

	// ErrFileLockedNotOwned the requested file is not locked by this peer
	ErrFileLockedNotOwned = errors.New("Request file is not locked by this peer")

	// ErrBucketAlreadyExists a bucket with the same name already exists in the current address book
	ErrBucketAlreadyExists = errors.New("Bucket already exists in local address book")

	// ErrBucketUnkown the requested bucket is not known
	ErrBucketUnkown = errors.New("Unkown bucket requested")

	// ErrBucketInvalidPrivateKey the requested bucket does not verify with the supplied private key
	ErrBucketInvalidPrivateKey = errors.New("the requested bucket does not verify with the supplied private key")
)

package interfaces

import (
	cid "github.com/ipfs/go-cid"
	pb "github.com/runletapp/crabfs/protos"
)

// EntryBookIterator used to iterate over a bucket
type EntryBookIterator interface {
	Next() *pb.CrabEntry
}

// EntryBook manages and stores crab entries
type EntryBook interface {
	Add(cid cid.Cid, entry *pb.CrabEntry) error

	Len() int

	GetByIndex(index uint) *pb.CrabEntry
	GetByCID(cid cid.Cid) *pb.CrabEntry

	Front() *pb.CrabEntry
	Back() *pb.CrabEntry

	Iter() EntryBookIterator
}

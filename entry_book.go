package crabfs

import (
	"container/heap"
	"sync"

	cid "github.com/ipfs/go-cid"
	"github.com/runletapp/crabfs/interfaces"
	pb "github.com/runletapp/crabfs/protos"
)

var _ interfaces.EntryBook = (*entryBookImpl)(nil)
var _ interfaces.EntryBookIterator = (*entryBookIteratorImpl)(nil)

type entryBookIteratorImpl struct {
	pos  uint
	book *entryBookImpl
}

type entryBookImpl struct {
	entries      *entriesHeap
	hashMap      map[string]*pb.CrabEntry
	entriesMutex sync.RWMutex
}

// EntryBookNew creates a new EntryBook
func EntryBookNew() (interfaces.EntryBook, error) {
	book := &entryBookImpl{
		entries: &entriesHeap{},
		hashMap: map[string]*pb.CrabEntry{},

		entriesMutex: sync.RWMutex{},
	}

	heap.Init(book.entries)

	return book, nil
}

func (book *entryBookImpl) Add(cid cid.Cid, entry *pb.CrabEntry) error {
	book.entriesMutex.Lock()
	defer book.entriesMutex.Unlock()

	_, prs := book.hashMap[cid.String()]
	if prs {
		// Already has it
		return nil
	}

	heap.Push(book.entries, entry)

	book.hashMap[cid.String()] = entry

	return nil
}

func (book *entryBookImpl) Len() int {
	book.entriesMutex.RLock()
	defer book.entriesMutex.RUnlock()

	return len(*book.entries)
}

func (book *entryBookImpl) GetByIndex(index uint) *pb.CrabEntry {
	book.entriesMutex.RLock()
	defer book.entriesMutex.RUnlock()

	if index >= uint(len(*book.entries)) {
		return nil
	}

	return (*book.entries)[index]
}

func (book *entryBookImpl) GetByCID(cid cid.Cid) *pb.CrabEntry {
	book.entriesMutex.RLock()
	defer book.entriesMutex.RUnlock()

	return book.hashMap[cid.String()]
}

func (book *entryBookImpl) Front() *pb.CrabEntry {
	return book.GetByIndex(0)
}

func (book *entryBookImpl) Back() *pb.CrabEntry {
	pos := uint(book.Len())
	if pos > 0 {
		pos = pos - 1
	}
	return book.GetByIndex(pos)
}

func (book *entryBookImpl) Iter() interfaces.EntryBookIterator {
	iter := &entryBookIteratorImpl{
		pos:  0,
		book: book,
	}

	return iter
}

func (iter *entryBookIteratorImpl) Next() *pb.CrabEntry {
	current := iter.pos
	iter.pos++

	return iter.book.GetByIndex(current)
}

package crabfs

import (
	"sync"

	"github.com/runletapp/crabfs/interfaces"
)

var _ interfaces.AddressBook = (*addresBookImpl)(nil)

type addresBookImpl struct {
	storageMutex sync.RWMutex
	storage      map[string]string
}

// AddressBookNew creates a new address book backed
func AddressBookNew() (interfaces.AddressBook, error) {
	book := &addresBookImpl{
		storageMutex: sync.RWMutex{},
		storage:      map[string]string{},
	}

	return book, nil
}

func (book *addresBookImpl) Add(name string, address string) error {
	book.storageMutex.Lock()
	defer book.storageMutex.Unlock()

	book.storage[name] = address

	return nil
}

func (book *addresBookImpl) Get(name string) (string, bool) {
	book.storageMutex.RLock()
	defer book.storageMutex.RUnlock()

	v, prs := book.storage[name]
	return v, prs
}

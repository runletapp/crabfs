package interfaces

// AddressBook stores bucket addresses
type AddressBook interface {
	Add(name string, address string) error

	Get(name string) (string, bool)
}

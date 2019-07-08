package identity

// Identity node interface
type Identity interface {
	GetPrivateKey() []byte

	// Marshal marshals a identity to bytes
	Marshal() ([]byte, error)
}

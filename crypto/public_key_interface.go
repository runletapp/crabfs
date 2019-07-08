package crypto

// PubKey public key abstraction
type PubKey interface {
	Marshal() ([]byte, error)
	Encrypt(data []byte, label []byte) ([]byte, error)
	Verify(data []byte, signature []byte) (bool, error)
	Hash() []byte
	HashString() string
}

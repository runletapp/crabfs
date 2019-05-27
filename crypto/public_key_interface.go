package crypto

import "io"

// PubKey public key abstraction
type PubKey interface {
	Marshall() (io.Reader, error)
	Encrypt(data []byte, label []byte) ([]byte, error)
	Validate(data []byte, signature []byte) (bool, error)
	Hash() []byte
}

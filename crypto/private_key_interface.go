package crypto

import (
	"io"
)

// PrivKey private key abstraction
type PrivKey interface {
	GetPublic() PubKey
	Marshall() (io.Reader, error)
	Decrypt(cipherText []byte, label []byte) ([]byte, error)
	Sign(data []byte) ([]byte, error)
	Hash() []byte
}

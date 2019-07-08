package crypto

// PrivKey private key abstraction
type PrivKey interface {
	GetPublic() PubKey
	Marshal() ([]byte, error)
	Decrypt(cipherText []byte, label []byte) ([]byte, error)
	Sign(data []byte) ([]byte, error)
	Hash() []byte
	HashString() string
}

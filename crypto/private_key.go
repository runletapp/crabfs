package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"io"
)

var _ PrivKey = &privKeyImpl{}

type privKeyImpl struct {
	internalPk *rsa.PrivateKey
	pub        PubKey

	hash []byte
}

// UnmarshalPrivateKey parse a private key from bytes generated with PrivKey.Marshal
func UnmarshalPrivateKey(b []byte) (PrivKey, error) {
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	return privateKeyFromRSA(key)
}

func privateKeyFromRSA(internalPk *rsa.PrivateKey) (PrivKey, error) {
	pk := &privKeyImpl{
		internalPk: internalPk,
	}

	hash := sha256.New()
	data, err := pk.Marshal()
	if err != nil {
		return nil, err
	}
	_, err = hash.Write(data)
	if err != nil {
		return nil, err
	}

	pk.hash = hash.Sum(nil)

	pk.pub, err = publicKeyNewFromRSA(internalPk.Public().(*rsa.PublicKey))
	if err != nil {
		return nil, err
	}

	return pk, nil
}

// PrivateKeyNew generate a new key
func PrivateKeyNew(random io.Reader, size int) (PrivKey, error) {
	internalPk, err := rsa.GenerateKey(random, size)
	if err != nil {
		return nil, err
	}

	return privateKeyFromRSA(internalPk)
}

func (pvk *privKeyImpl) GetPublic() PubKey {
	return pvk.pub
}

func (pvk *privKeyImpl) Marshal() ([]byte, error) {
	data := x509.MarshalPKCS1PrivateKey(pvk.internalPk)

	return data, nil
}

func (pvk *privKeyImpl) Decrypt(cipherText []byte, label []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, pvk.internalPk, cipherText, label)
}

func (pvk *privKeyImpl) Sign(data []byte) ([]byte, error) {
	hashed := sha256.Sum256(data)
	return rsa.SignPSS(rand.Reader, pvk.internalPk, crypto.SHA256, hashed[:], nil)
}

func (pvk *privKeyImpl) Hash() []byte {
	return pvk.hash
}

func (pvk *privKeyImpl) HashString() string {
	return hex.EncodeToString(pvk.hash)
}

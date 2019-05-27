package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"io"
	"io/ioutil"
)

var _ PubKey = &publicKeyImpl{}

type publicKeyImpl struct {
	internalPk *rsa.PublicKey

	hash []byte
}

// UnmarshallPublicKey parse a public key from bytes generated with PubKey.Marshall
func UnmarshallPublicKey(r io.Reader) (PubKey, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS1PublicKey(data)
	if err != nil {
		return nil, err
	}

	return publicKeyNewFromRSA(key)
}

func publicKeyNewFromRSA(pub *rsa.PublicKey) (PubKey, error) {
	pk := &publicKeyImpl{
		internalPk: pub,
	}

	hash := sha256.New()
	data, err := pk.Marshall()
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(hash, data)
	if err != nil {
		return nil, err
	}

	pk.hash = hash.Sum(nil)

	return pk, nil
}

func (puk *publicKeyImpl) Marshall() (io.Reader, error) {
	data := x509.MarshalPKCS1PublicKey(puk.internalPk)
	return bytes.NewReader(data), nil
}

func (puk *publicKeyImpl) Encrypt(data []byte, label []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, puk.internalPk, data, label)
}

func (puk *publicKeyImpl) Validate(data []byte, signature []byte) (bool, error) {
	hashed := sha256.Sum256(data)
	err := rsa.VerifyPSS(puk.internalPk, crypto.SHA256, hashed[:], signature, nil)
	return err == nil, err
}

func (puk *publicKeyImpl) Hash() []byte {
	return puk.hash
}

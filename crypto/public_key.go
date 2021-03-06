package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"

	"github.com/gogo/protobuf/proto"

	"github.com/runletapp/crabfs/crypto/protos"
)

var _ PubKey = &publicKeyImpl{}

type publicKeyImpl struct {
	internalPk *rsa.PublicKey

	hash []byte
}

// UnmarshalPublicKey parse a public key from bytes generated with PubKey.Marshal
func UnmarshalPublicKey(b []byte) (PubKey, error) {
	var pbKey protos.Key
	if err := proto.Unmarshal(b, &pbKey); err != nil {
		return nil, err
	}

	// We only support x509 for now

	key, err := x509.ParsePKCS1PublicKey(pbKey.Data)
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
	data, err := pk.Marshal()
	if err != nil {
		return nil, err
	}
	_, err = hash.Write(data)
	if err != nil {
		return nil, err
	}

	pk.hash = hash.Sum(nil)

	return pk, nil
}

func (puk *publicKeyImpl) Marshal() ([]byte, error) {
	data := x509.MarshalPKCS1PublicKey(puk.internalPk)

	pb := protos.Key{
		Format: protos.Key_x509,
		Data:   data,
	}

	data, err := proto.Marshal(&pb)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (puk *publicKeyImpl) Encrypt(data []byte, label []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, puk.internalPk, data, label)
}

func (puk *publicKeyImpl) Verify(data []byte, signature []byte) (bool, error) {
	hashed := sha256.Sum256(data)
	err := rsa.VerifyPSS(puk.internalPk, crypto.SHA256, hashed[:], signature, nil)
	return err == nil, err
}

func (puk *publicKeyImpl) Hash() []byte {
	return puk.hash
}

func (puk *publicKeyImpl) HashString() string {
	return hex.EncodeToString(puk.hash)
}

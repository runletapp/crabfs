package cryptostream

// This is heavily inspired by https://github.com/libp2p/go-libp2p-pnet/blob/master/psk_conn.go

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	salsa20 "github.com/davidlazar/go-crypto/salsa20"
	pool "github.com/libp2p/go-buffer-pool"
	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	ipnet "github.com/libp2p/go-libp2p-interface-pnet"
	libp2pNet "github.com/libp2p/go-libp2p-net"
	"github.com/multiformats/go-multihash"
)

// we are using buffer pool as user needs their slice back
// so we can't do XOR cripter in place
var (
	errShortNonce  = ipnet.NewError("could not read full nonce")
	errInsecureNil = ipnet.NewError("insecure is nil")
	errPSKNil      = ipnet.NewError("pre-shread key is nil")
)

// CryptoStream encrypt/decrypt all data that flows into this stream
type CryptoStream struct {
	io.ReadWriteCloser

	rawStream libp2pNet.Stream

	psk *[32]byte

	writeS20 cipher.Stream
	readS20  cipher.Stream
}

func (c *CryptoStream) Read(out []byte) (int, error) {
	if c.readS20 == nil {
		nonce := make([]byte, 24)
		_, err := io.ReadFull(c.rawStream, nonce)
		if err != nil {
			return 0, errShortNonce
		}
		c.readS20 = salsa20.New(c.psk, nonce)
	}

	n, err := c.rawStream.Read(out) // read to in
	if n > 0 {
		c.readS20.XORKeyStream(out[:n], out[:n]) // decrypt to out buffer
	}
	return n, err
}

func (c *CryptoStream) Write(in []byte) (int, error) {
	if c.writeS20 == nil {
		nonce := make([]byte, 24)
		_, err := rand.Read(nonce)
		if err != nil {
			return 0, err
		}
		_, err = c.rawStream.Write(nonce)
		if err != nil {
			return 0, err
		}

		c.writeS20 = salsa20.New(c.psk, nonce)
	}
	out := pool.Get(len(in))
	defer pool.Put(out)

	c.writeS20.XORKeyStream(out, in) // encrypt

	return c.rawStream.Write(out) // send
}

var _ io.ReadWriteCloser = (*CryptoStream)(nil)

// New creates a new crypto channel
func New(privateKey *libp2pCrypto.RsaPrivateKey, insecure libp2pNet.Stream) (io.ReadWriteCloser, error) {
	if insecure == nil {
		return nil, errInsecureNil
	}
	if privateKey == nil {
		return nil, errPSKNil
	}

	var psk [32]byte

	data, err := privateKey.Bytes()
	if err != nil {
		return nil, err
	}

	pskHash, err := multihash.Sum(data, multihash.SHA3_256, -1)
	if err != nil {
		return nil, err
	}
	copy(psk[:], []byte(pskHash.String())[:32])

	return &CryptoStream{
		rawStream: insecure,
		psk:       &psk,
	}, nil
}

// Close closes the current stream and the underlying insecure stream
func (c *CryptoStream) Close() error {
	return c.rawStream.Close()
}

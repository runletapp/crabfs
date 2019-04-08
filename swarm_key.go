package crabfs

// Heavily inspired by https://github.com/libp2p/go-libp2p-pnet

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

var (
	pathPSKv1  = []byte("/key/swarm/psk/1.0.0/")
	pathBin    = "/bin/"
	pathBase16 = "/base16/"
	pathBase64 = "/base64/"
)

func readHeader(r *bufio.Reader) ([]byte, error) {
	header, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return bytes.TrimRight(header, "\r\n"), nil
}

func expectHeader(r *bufio.Reader, expected []byte) error {
	header, err := readHeader(r)
	if err != nil {
		return err
	}
	if !bytes.Equal(header, expected) {
		return fmt.Errorf("expected file header %s, got: %s", pathPSKv1, header)
	}
	return nil
}

func newLine() io.Reader {
	return bytes.NewReader([]byte("\n"))
}

// GenerateSwarmKey generates a private key ready to be used in the swarm protector
func GenerateSwarmKey() (io.Reader, error) {
	psk := [32]byte{}
	_, err := rand.Read(psk[:])
	if err != nil {
		return nil, err
	}

	hexPsk := make([]byte, len(psk)*2)
	hex.Encode(hexPsk, psk[:])

	// just a shortcut to NewReader
	nr := func(b []byte) io.Reader {
		return bytes.NewReader(b)
	}
	return io.MultiReader(nr(pathPSKv1), newLine(), nr([]byte("/base16/")), newLine(), nr(hexPsk)), nil
}

// ReadSwarmKey reads the key from a reader
func ReadSwarmKey(in io.Reader) (*[32]byte, error) {
	reader := bufio.NewReader(in)
	if err := expectHeader(reader, pathPSKv1); err != nil {
		return nil, err
	}
	header, err := readHeader(reader)
	if err != nil {
		return nil, err
	}

	var decoder io.Reader
	switch string(header) {
	case pathBase16:
		decoder = hex.NewDecoder(reader)
	case pathBase64:
		decoder = base64.NewDecoder(base64.StdEncoding, reader)
	case pathBin:
		decoder = reader
	default:
		return nil, fmt.Errorf("unknown encoding: %s", header)
	}
	out := new([32]byte)
	_, err = io.ReadFull(decoder, out[:])
	if err != nil {
		return nil, err
	}
	return out, nil
}

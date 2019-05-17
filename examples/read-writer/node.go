package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/runletapp/crabfs/interfaces"
	"github.com/runletapp/crabfs/options"

	"github.com/runletapp/crabfs"
)

func nodeStart(ctx context.Context, bootstrapAddr string, mountLocation string, privateKey io.Reader) interfaces.Core {
	fs, err := crabfs.New(
		options.Root(mountLocation),
		options.Context(ctx),
		options.BucketName("exampleBkt"),
		options.BootstrapPeers([]string{bootstrapAddr}),
		options.PrivateKey(privateKey),
	)
	if err != nil {
		panic(err)
	}

	log.Printf("Host id: %s\n", fs.GetID())
	for i, addr := range fs.GetAddrs() {
		log.Printf("Host addr [%d]: %s\n", i, addr)
	}

	return fs
}

func reader(ctx context.Context, fs interfaces.Core, filename string) {
	log.Printf("Looking for: %s", filename)

	file, size, err := fs.Get(ctx, filename)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Size: %d", size)

	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		log.Printf("Seek Err: %v", err)
	}

	_, err = io.Copy(os.Stdout, file)
	if err != nil {
		log.Printf("Copy Err: %v", err)
	}
}

func writer(ctx context.Context, fs interfaces.Core, filename string) {
	file := &bytes.Buffer{}

	log.Printf("Reading from stdin. Press Ctrl+D to stop and close")

	buffer := make([]byte, 500)
	for {
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			break
		}

		_, err = file.Write(buffer[:n])
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Saving...")

	if err := fs.Put(ctx, filename, file, time.Now()); err != nil {
		panic(err)
	}

	log.Printf("Done.")
}

func main() {
	outputFile := flag.String("o", "", "Output file")
	privateKeyFile := flag.String("p", "", "Private key file")
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	mountLocation := flag.String("m", "tmp/mount", "mount location")
	readFile := flag.String("q", "", "read file")
	writeFile := flag.String("w", "", "write file")
	flag.Parse()

	ctx := context.Background()

	if *outputFile != "" {
		log.Printf("Generating new private key...")
		file, err := os.Create(*outputFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		generator, err := crabfs.GenerateKeyPair()
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(file, generator)
		if err != nil {
			panic(err)
		}
		return
	}

	log.Printf("Starting node...")

	var psk io.Reader
	if *privateKeyFile != "" {
		pskFile, err := os.Open(*privateKeyFile)
		if err != nil {
			panic(err)
		}
		defer pskFile.Close()
		psk = pskFile
	}

	fs := nodeStart(ctx, *bootstrapPeer, *mountLocation, psk)

	if *readFile != "" {
		reader(ctx, fs, *readFile)
	} else if *writeFile != "" {
		writer(ctx, fs, *writeFile)
	}

	<-ctx.Done()
}

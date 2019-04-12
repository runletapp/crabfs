package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/runletapp/crabfs/options"

	"github.com/runletapp/crabfs"
)

func nodeStart(ctx context.Context, bootstrapAddr string, mountLocation string, privateKey io.Reader) *crabfs.CrabFS {
	fs, err := crabfs.New(
		mountLocation,
		options.Context(ctx),
		options.BucketName("exampleBkt"),
		options.BootstrapPeers([]string{bootstrapAddr}),
		options.PrivateKey(privateKey),
	)
	if err != nil {
		panic(err)
	}

	log.Printf("Host id: %s\n", fs.GetHostID())
	for i, addr := range fs.GetAddrs() {
		log.Printf("Host addr [%d]: %s\n", i, addr)
	}

	return fs
}

func reader(ctx context.Context, fs *crabfs.CrabFS, filename string) {
	log.Printf("Looking for: %s", filename)
	upstreamRecord, err := fs.GetContentRecord(ctx, filename)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Found: %v", upstreamRecord.ContentID.String())
	log.Printf("Size: %v", upstreamRecord.Length)
	log.Printf("Looking for providers of: %v", upstreamRecord.ContentID.String())

	providers, err := fs.GetProviders(ctx, upstreamRecord.ContentID)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for i, addrs := range providers {
		log.Printf("Provider: %d", i+1)
		for ia, addr := range addrs {
			log.Printf("Addr [%d]: %s", ia, addr)
		}
	}

	file, err := fs.Open(filename)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Waiting file to propagate")
	<-time.After(1 * time.Second)

	if err := file.Close(); err != nil {
		log.Printf("Error: %v", err)
	}
	log.Printf("Closed")
}

func writer(ctx context.Context, fs *crabfs.CrabFS, filename string) {
	file, err := fs.Create(filename)
	if err != nil {
		panic(err)
	}

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

	if err := file.Close(); err != nil {
		panic(err)
	}

	log.Printf("Done.")
}

func relayStart(ctx context.Context) {
	relay, err := crabfs.RelayNew(ctx, 1717, []string{})
	if err != nil {
		panic(err)
	}

	log.Printf("Relay id: %s\n", relay.GetRelayID())
	for i, addr := range relay.GetAddrs() {
		log.Printf("Relay addr [%d]: %s\n", i, addr)
	}
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

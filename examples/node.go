package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/src-d/go-billy.v4/osfs"

	"github.com/runletapp/crabfs"
	"gopkg.in/src-d/go-billy.v4"
)

func nodeStart(ctx context.Context, discoveryKey string, bootstrapAddr string, mountFS billy.Filesystem) *crabfs.CrabFS {
	fs, err := crabfs.NewWithContext(ctx, "exampleBkt", discoveryKey, 0, mountFS)
	if err != nil {
		panic(err)
	}
	if err := fs.Bootstrap([]string{bootstrapAddr}); err != nil {
		panic(err)
	}

	log.Printf("Host id: %s\n", fs.GetHostID())
	for i, addr := range fs.GetAddrs() {
		log.Printf("Host addr [%d]: %s\n", i, addr)
	}

	return fs
}

func reader(ctx context.Context, fs *crabfs.CrabFS, filename string) {
	<-time.After(5 * time.Second)
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
	relay, err := crabfs.RelayNew(ctx, 1717)
	if err != nil {
		panic(err)
	}

	log.Printf("Relay id: %s\n", relay.GetID())
	localAddr := ""
	for i, addr := range relay.GetAddrs() {
		log.Printf("Relay addr [%d]: %s\n", i, addr)
		if strings.HasPrefix(addr, "/ip4/127") {
			localAddr = addr
		}
	}

	mountFS := osfs.New("/tmp")

	nodeStart(ctx, "", localAddr, mountFS)
}

func main() {
	discoveryKey := flag.String("k", "example", "discovery key")
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	mountLocation := flag.String("m", "tmp/mount", "mount location")
	readFile := flag.String("q", "", "read file")
	writeFile := flag.String("w", "", "write file")
	relayFlag := flag.Bool("relay", false, "Start a relay instead of a node")
	flag.Parse()

	ctx := context.Background()

	if *relayFlag {
		log.Printf("Starting relay...")
		relayStart(ctx)
		<-ctx.Done()
		return
	}

	log.Printf("Starting node...")
	fs := nodeStart(ctx, *discoveryKey, *bootstrapPeer, osfs.New(*mountLocation))

	if err := fs.Announce(ctx); err != nil {
		panic(err)
	}

	if *readFile != "" {
		reader(ctx, fs, *readFile)
	} else if *writeFile != "" {
		writer(ctx, fs, *writeFile)
	}

	<-ctx.Done()
}

package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"gitlab.com/runletapp/crabfs"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

func nodeStart(discoveryKey string, bootstrapAddr string, mountLocation string) {
	ctx := context.Background()

	tmpfs := memfs.New()

	fs, err := crabfs.NewWithContext(ctx, "exampleBkt", mountLocation, discoveryKey, 0, tmpfs)
	if err != nil {
		panic(err)
	}
	if err := fs.Bootstrap([]string{bootstrapAddr}); err != nil {
		panic(err)
	}

	if err := fs.Announce(); err != nil {
		panic(err)
	}

	log.Printf("Host id: %s\n", fs.GetHostID())
	for i, addr := range fs.GetAddrs() {
		log.Printf("Host addr [%d]: %s\n", i, addr)
	}

	if mountLocation != "" {
		go func() {
			<-time.After(5 * time.Second)
			log.Printf("Looking for: test.txt")
			cid, err := fs.GetContentID(ctx, "test.txt")
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}

			log.Printf("Found: %v", cid.String())
			log.Printf("Looking for providers of: %v", cid.String())

			providers, err := fs.GetProviders(ctx, cid)
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}

			for i, addrs := range providers {
				log.Printf("Provider: %d", i)
				for ia, addr := range addrs {
					log.Printf("Addr [%d]: %s", ia, addr)
				}
			}
		}()
	}

	<-ctx.Done()
}

func relayStart() {
	ctx := context.Background()

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

	nodeStart("", localAddr, "")
}

func main() {
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	mountLocation := flag.String("m", "tmp/mount", "mount location")
	relayFlag := flag.Bool("relay", false, "Start a relay instead of a node")
	flag.Parse()

	if *relayFlag {
		log.Printf("Starting relay...")
		relayStart()
	} else {
		log.Printf("Starting node...")
		nodeStart("example", *bootstrapPeer, *mountLocation)
	}
}

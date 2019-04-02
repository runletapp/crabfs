package main

import (
	"context"
	"flag"
	"log"

	"gitlab.com/runletapp/crabfs"
)

func nodeStart(bootstrapAddr string) {
	ctx := context.Background()

	fs, err := crabfs.NewWithContext(ctx, "tmp/mount", "test")
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

	<-ctx.Done()
}

func relayStart() {
	ctx := context.Background()

	relay, err := crabfs.RelayNew(ctx, 1717)
	if err != nil {
		panic(err)
	}

	log.Printf("Relay id: %s\n", relay.GetID())
	for i, addr := range relay.GetAddrs() {
		log.Printf("Relay addr [%d]: %s\n", i, addr)
	}

	<-ctx.Done()
}

func main() {
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	relayFlag := flag.Bool("relay", false, "Start a relay instead of a node")
	flag.Parse()

	if *relayFlag {
		log.Printf("Starting relay...")
		relayStart()
	} else {
		log.Printf("Starting node...")
		nodeStart(*bootstrapPeer)
	}
}

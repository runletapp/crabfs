package main

import (
	"context"
	"flag"
	"log"

	"github.com/runletapp/crabfs"
)

func relayStart(ctx context.Context, bootstrapPeers []string) {
	relay, err := crabfs.RelayNew(ctx, 1717, bootstrapPeers)
	if err != nil {
		panic(err)
	}

	log.Printf("Relay id: %s\n", relay.GetRelayID())
	for i, addr := range relay.GetAddrs() {
		log.Printf("Relay addr [%d]: %s\n", i, addr)
	}
}

func main() {
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	flag.Parse()

	ctx := context.Background()

	log.Printf("Starting relay...")
	relayStart(ctx, []string{*bootstrapPeer})
	<-ctx.Done()
}

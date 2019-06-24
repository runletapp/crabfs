package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/runletapp/crabfs/identity"

	"github.com/runletapp/crabfs"
)

func relayStart(ctx context.Context, bootstrapPeers []string, id identity.Identity) {
	relay, err := crabfs.RelayNew(ctx, 1700, bootstrapPeers, id)
	if err != nil {
		panic(err)
	}

	log.Printf("ID: %s\n", relay.GetHostID())
	for i, addr := range relay.GetAddrs() {
		log.Printf("Addr [%d]: %s\n", i, addr)
	}
}

func main() {
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	idFile := flag.String("i", "", "identity file")
	flag.Parse()

	ctx := context.Background()

	bootstrapPeers := []string{}
	if strings.TrimSpace(*bootstrapPeer) != "" {
		bootstrapPeers = append(bootstrapPeers, *bootstrapPeer)
	}

	var id identity.Identity
	if strings.TrimSpace(*idFile) != "" {
		file, err := os.Open(*idFile)
		if err != nil {
			id, err = identity.CreateIdentity()
			if err != nil {
				panic(err)
			}
			data, err := id.Marshal()
			if err != nil {
				panic(err)
			}
			if err := ioutil.WriteFile(*idFile, data, 0644); err != nil {
				panic(err)
			}
		} else {
			defer file.Close()

			id, err = identity.UnmarshalIdentity(file)
			if err != nil {
				panic(err)
			}
		}
	}

	log.Printf("Starting relay...")
	relayStart(ctx, bootstrapPeers, id)
	<-ctx.Done()
}

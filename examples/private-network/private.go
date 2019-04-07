package main

import "flag"

func main() {
	outputfile := flag.String("o", "private-key.psk", "Outpuf file")
	bootstrapPeer := flag.String("d", "", "bootstrap peer to dial")
	mountLocation := flag.String("m", "tmp/mount", "mount location")
	readFile := flag.String("q", "", "read file")
	writeFile := flag.String("w", "", "write file")
	relayFlag := flag.Bool("relay", false, "Start a relay instead of a node")
	flag.Parse()
}

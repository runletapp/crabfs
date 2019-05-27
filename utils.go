package crabfs

import (
	"fmt"

	multihash "github.com/multiformats/go-multihash"
)

// KeyFromFilename converts a file name to a key
func KeyFromFilename(pk string, bucketFilename string) string {
	hash, _ := multihash.Sum([]byte(bucketFilename), multihash.SHA3_256, -1)
	return fmt.Sprintf("/crabfs/v1/%s/%s", pk, hash.String())
}

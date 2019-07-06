package crabfs

import (
	"strings"

	pb "github.com/runletapp/crabfs/protos"
)

type entriesHeap []*pb.CrabEntry

func (h entriesHeap) Len() int           { return len(h) }
func (h entriesHeap) Less(i, j int) bool { return strings.Compare(h[i].Timestamp, h[j].Timestamp) < 0 }
func (h entriesHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *entriesHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*pb.CrabEntry))
}

func (h *entriesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

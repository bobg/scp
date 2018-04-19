// +build ignore

package main

import (
	"flag"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bobg/scp"
)

type entry struct {
	node *scp.Node
	ch   chan *scp.Env
}

type nodeIDType int

func (n nodeIDType) String() string {
	return strconv.Itoa(n)
}

// Usage:
//   go run toy.go [-seed N] '2 3 4 / 2 3 5 / 6 7 8' '1 3 4 / 7 8' ...
// Each argument describes the quorum slices for the corresponding node (1-based).
// Nodes do not specify themselves as quorum slice members.

func main() {
	seed := flag.Int64("seed", 1, "RNG seed")
	flag.Parse()
	rand.Seed(*seed)

	entries := []entry{nil} // nodes are numbered starting at 1

	ch := make(chan *scp.Env)
	var highestSlot slotIDType
	for i, arg := range flag.Args() {
		nodeID := nodeIDType(i + 1)
		var q scp.QSet
		for _, slices := range strings.Split(arg, "/") {
			var qslice []scp.NodeID
			for _, field := range strings.Fields(slice) {
				f, err := strconv.Atoi(field)
				if err != nil {
					log.Fatal(err)
				}
				if f <= 0 || f == nodeID {
					log.Print("skipping quorum slice member %d for node %d", f, nodeID)
					continue
				}
				qslice = append(qslice, nodeID)
			}
			q = append(q, qslice)
		}
		nodeCh := make(chan *scp.Env)
		e := entry{node: scp.NewNode(nodeID, &q), ch: nodeCh}
		entries = append(entries, e)
		go nodefn(e.node, e.ch, ch, &highestSlot)
	}

	for env := range ch {
		if env.I > highestSlot { // this is the only thread that writes highestSlot, so it's ok to read it non-atomically
			atomic.StoreInt32(&highestSlot, env.I)
		}

		// Send this message to each of the node's peers.
		nodeID := env.V.(nodeIDType)
		for _, peerID := range entries[nodeID].node.Peers() {
			nodes[peerID.(nodeIDType)].ch <- env
		}
	}
}

const (
	minNomDelayMS = 500
	maxNomDelayMS = 2000
)

// runs as a goroutine
func nodefn(n *scp.Node, recv <-chan *scp.Env, send chan<- *scp.Env, highestSlot *slotIDType) {
	for {
		// Some time in the next minNomDelayMS to maxNomDelayMS
		// milliseconds, nominate a value for a new slot.
		timeCh := make(chan struct{})
		timer := time.AfterFunc((minNomDelayMS+rand.Intn(maxNomDelayMS-minNomDelayMS))*time.Millisecond, func() { close(timech) })

		select {
		case env := <-recv:
			// Never mind about the nomination timer.
			timer.Stop()
			close(timeCh)

			n.Handle(env, send)

		case <-timeCh:
			val := valType(rand.Intn(20))
			slotID := 1 + atomic.LoadInt32(highestSlot)

			// Send a nominate message "from" the node to itself. If it has
			// max priority among its neighbors (for this slot) it will
			// propagate the nomination.
			vs := new(scp.ValueSet)
			vs.Add(val)
			env := &scp.Env{
				V: n.ID,
				I: slotID,
				Q: n.Q,
				M: &scp.NomMsg{
					X: vs,
				},
			}
			n.Handle(env, send)
		}
	}
}

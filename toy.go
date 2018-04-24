// +build ignore

package main

// Usage:
//   go run toy.go [-seed N] 'alice: bob carol david / bob carol ed / fran gabe hank' 'bob: alice carol david / gabe hank' ...
// Each argument gives a node's name (before the colon) and the node's
// quorum slices.
// Nodes do not specify themselves as quorum slice members, though
// they are understood to belong to every quorum slice.

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"log"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bobg/scp"
	"golang.org/x/time/rate"
)

type entry struct {
	node *scp.Node
	ch   chan *scp.Msg
}

type valType string

func (v valType) Less(other scp.Value) bool {
	return v < other.(valType)
}

func (v valType) Combine(other scp.Value) scp.Value {
	if v < other.(valType) {
		return v
	}
	return other
}

func (v valType) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, v)
	return buf.Bytes()
}

func (v valType) String() string {
	return string(v)
}

func main() {
	seed := flag.Int64("seed", 1, "RNG seed")
	flag.Parse()
	rand.Seed(*seed)

	entries := make(map[scp.NodeID]entry)

	ch := make(chan *scp.Msg, 10000)
	var highestSlot int32
	for _, arg := range flag.Args() {
		parts := strings.SplitN(arg, ":", 2)
		nodeID := scp.NodeID(parts[0])
		var q []scp.NodeIDSet
		for _, slices := range strings.Split(parts[1], "/") {
			var qslice scp.NodeIDSet
			for _, field := range strings.Fields(slices) {
				if field == string(nodeID) {
					log.Print("skipping quorum slice member %s for node %s", field, nodeID)
					continue
				}
				qslice = qslice.Add(scp.NodeID(field))
			}
			q = append(q, qslice)
		}
		node := scp.NewNode(nodeID, q)
		nodeCh := make(chan *scp.Msg, 1000)
		entries[nodeID] = entry{node: node, ch: nodeCh}
		go nodefn(node, nodeCh, ch, &highestSlot)
	}

	for msg := range ch {
		if _, ok := msg.T.(*scp.NomTopic); !ok && int32(msg.I) > highestSlot { // this is the only thread that writes highestSlot, so it's ok to read it non-atomically
			atomic.StoreInt32(&highestSlot, int32(msg.I))
			log.Printf("highestSlot is now %d", highestSlot)
		}

		// Send this message to every other node.
		for nodeID, entry := range entries {
			if nodeID == msg.V {
				continue
			}
			entry.ch <- msg
		}
	}
}

const (
	minNomDelayMS = 500
	maxNomDelayMS = 2000
)

// runs as a goroutine
func nodefn(n *scp.Node, recv <-chan *scp.Msg, send chan<- *scp.Msg, highestSlot *int32) {
	limiter := rate.NewLimiter(10, 10)
	for {
		// Some time in the next minNomDelayMS to maxNomDelayMS
		// milliseconds.
		timer := time.NewTimer(time.Duration((minNomDelayMS + rand.Intn(maxNomDelayMS-minNomDelayMS)) * int(time.Millisecond)))

		select {
		case msg := <-recv:
			// Never mind about the nomination timer.
			if !timer.Stop() {
				<-timer.C
			}

			limiter.Wait(context.Background())

			res, err := n.Handle(msg)
			if err != nil {
				n.Logf("could not handle %s: %s", msg, err)
				continue
			}
			if res != nil {
				n.Logf("handled %s -> %s", msg, res)
				send <- res
			}

		case <-timer.C:
			// xxx should acquire n.mu
			var prodded bool
			peers := n.Peers()
			for _, slot := range n.Pending {
				if slot.Ph != scp.PhNom {
					continue
				}
				for _, peer := range peers {
					if msg, ok := slot.M[peer]; ok {
						res, err := n.Handle(msg)
						if err != nil {
							n.Logf("error prodding node with %s: %s", msg, err)
						} else if res != nil {
							send <- res
						}
						prodded = true
					}
				}
			}

			if prodded {
				n.Logf("prodded")
				continue
			}

			slotID := 1 + scp.SlotID(atomic.LoadInt32(highestSlot))
			val := foods[rand.Intn(len(foods))]

			// Send a nominate message "from" the node to itself. If it has
			// max priority among its neighbors (for this slot) it will
			// propagate the nomination.
			var vs scp.ValueSet
			vs = vs.Add(val)
			msg := scp.NewMsg(n.ID, scp.SlotID(slotID), n.Q, &scp.NomTopic{X: vs})
			n.Logf("trying to get something started with %s", msg)
			res, err := n.Handle(msg)
			if err != nil {
				n.Logf("could not handle %s: %s", msg, err)
				continue
			}
			if res != nil {
				send <- res
			}
		}
	}
}

var foods = []valType{
	"pizza",
	"burgers",
	"burritos",
	"sandwiches",
	"sushi",
	"salads",
	"gyros",
	"indian",
	"soup",
	"pasta",
}

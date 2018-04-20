// +build ignore

package main

import (
	"bytes"
	"encoding/binary"
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

type valType int

func (v valType) Less(other scp.Value) bool {
	return v < other.(valType)
}

func (v valType) Combine(other scp.Value) scp.Value {
	return valType(v + other.(valType))
}

func (v valType) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, v)
	return buf.Bytes()
}

func (v valType) String() string {
	return strconv.Itoa(int(v))
}

// Usage:
//   go run toy.go [-seed N] '2 3 4 / 2 3 5 / 6 7 8' '1 3 4 / 7 8' ...
// Each argument describes the quorum slices for the corresponding node (1-based).
// Nodes do not specify themselves as quorum slice members.

func main() {
	seed := flag.Int64("seed", 1, "RNG seed")
	flag.Parse()
	rand.Seed(*seed)

	entries := make(map[scp.NodeID]entry)

	ch := make(chan *scp.Env, 5)
	var highestSlot int32
	for i, arg := range flag.Args() {
		nodeID := scp.NodeID(strconv.Itoa(i + 1))
		var q [][]scp.NodeID
		for _, slices := range strings.Split(arg, "/") {
			var qslice []scp.NodeID
			for _, field := range strings.Fields(slices) {
				if field == string(nodeID) {
					log.Print("skipping quorum slice member %s for node %s", field, nodeID)
					continue
				}
				qslice = append(qslice, scp.NodeID(field))
			}
			q = append(q, qslice)
		}
		nodeCh := make(chan *scp.Env)
		e := entry{node: scp.NewNode(nodeID, q), ch: nodeCh}
		entries[nodeID] = e
		go nodefn(e.node, e.ch, ch, &highestSlot)
	}

	// log.Print(spew.Sdump(entries))

	for env := range ch {
		if _, ok := env.M.(*scp.NomMsg); !ok && int32(env.I) > highestSlot { // this is the only thread that writes highestSlot, so it's ok to read it non-atomically
			atomic.StoreInt32(&highestSlot, int32(env.I))
			log.Printf("highestSlot is now %d", highestSlot)
		}

		peers := entries[env.V].node.Peers()
		log.Printf("main: dispatching to %s, msg: %s", peers, env)

		// Send this message to each of the node's peers.
		for _, peerID := range peers {
			entries[peerID].ch <- env
		}
	}
}

const (
	minNomDelayMS = 500
	maxNomDelayMS = 2000
)

// runs as a goroutine
func nodefn(n *scp.Node, recv <-chan *scp.Env, send chan<- *scp.Env, highestSlot *int32) {
	for {
		// Some time in the next minNomDelayMS to maxNomDelayMS
		// milliseconds, nominate a value for a new slot.
		timeCh := make(chan struct{})
		timer := time.AfterFunc(time.Duration((minNomDelayMS+rand.Intn(maxNomDelayMS-minNomDelayMS))*int(time.Millisecond)), func() { close(timeCh) })

		select {
		case env := <-recv:
			// Never mind about the nomination timer.
			timer.Stop()
			close(timeCh)

			res, err := n.Handle(env)
			if err != nil {
				n.Logf("could not handle %s: %s", env, err)
				continue
			}
			if res == nil {
				n.Logf("ignored %s", env)
			} else {
				n.Logf("handled %s -> %s", env, res)
				send <- res
			}

		case <-timeCh:
			val := valType(rand.Intn(20))
			slotID := 1 + atomic.LoadInt32(highestSlot)

			// Send a nominate message "from" the node to itself. If it has
			// max priority among its neighbors (for this slot) it will
			// propagate the nomination.
			var vs scp.ValueSet
			vs.Add(val)
			env := &scp.Env{
				V: n.ID,
				I: scp.SlotID(slotID),
				Q: n.Q,
				M: &scp.NomMsg{
					X: vs,
				},
			}
			n.Logf("trying to get something started with %s", env)
			res, err := n.Handle(env)
			if err != nil {
				n.Logf("could not handle %s: %s", env, err)
				continue
			}
			if res != nil {
				send <- res
			}
		}
	}
}

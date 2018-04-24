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
	"encoding/binary"
	"flag"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/bobg/scp"
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
		go nodefn(node, nodeCh, ch)
	}

	for slotID := scp.SlotID(1); ; slotID++ {
		topics := make(map[scp.NodeID]scp.Topic)

		for _, e := range entries {
			e.ch <- nil // a nil message means "start a new slot"
			topics[e.node.ID] = nil
		}
		for msg := range ch {
			if msg.I < slotID {
				// discard messages about old slots
				continue
			}

			n := entries[msg.V].node
			if topics[msg.V] == nil {
				n.Logf("%s", msg)
			} else {
				switch topics[msg.V].(type) {
				case *scp.NomTopic:
					if _, ok := msg.T.(*scp.PrepTopic); ok {
						n.Logf("%s", msg)
					}
				case *scp.PrepTopic:
					if _, ok := msg.T.(*scp.CommitTopic); ok {
						n.Logf("%s", msg)
					}
				case *scp.CommitTopic:
					if _, ok := msg.T.(*scp.ExtTopic); ok {
						n.Logf("%s", msg)
					}
				}
			}
			topics[msg.V] = msg.T

			allExt := true
			for _, topic := range topics {
				if _, ok := topic.(*scp.ExtTopic); !ok {
					allExt = false
					break
				}
			}
			if allExt {
				log.Print("all externalized")
				break
			}

			// Send this message to every other node.
			for nodeID, e := range entries {
				if nodeID == msg.V {
					continue
				}
				e.ch <- msg
			}
		}
	}
}

// runs as a goroutine
func nodefn(n *scp.Node, recv <-chan *scp.Msg, send chan<- *scp.Msg) {
	for {
		// Prod the node after a second of inactivity.
		timer := time.NewTimer(time.Second)

		select {
		case msg := <-recv:
			if msg == nil {
				// New round, try to nominate something.
				var slotID scp.SlotID
				for i := range n.Ext {
					if i > slotID {
						slotID = i
					}
				}
				slotID++
				val := foods[rand.Intn(len(foods))]
				msg = scp.NewMsg(n.ID, slotID, n.Q, &scp.NomTopic{X: scp.ValueSet{val}})
			}

			// Never mind about the prodding timer.
			if !timer.Stop() {
				<-timer.C
			}

			res, err := n.Handle(msg)
			if err != nil {
				n.Logf("could not handle %s: %s", msg, err)
				continue
			}
			if res != nil {
				// n.Logf("handled %s -> %s", msg, res)
				send <- res
			}

		case <-timer.C:
			// xxx should acquire n.mu
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
							// n.Logf("prodded with %s -> %s", msg, res)
							send <- res
						}
					}
				}
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

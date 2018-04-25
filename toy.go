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
		node := scp.NewNode(nodeID, q, ch)
		nodeCh := make(chan *scp.Msg, 1000)
		entries[nodeID] = entry{node: node, ch: nodeCh}
		go nodefn(node, nodeCh)
	}

	for slotID := scp.SlotID(1); ; slotID++ {
		msgs := make(map[scp.NodeID]*scp.Msg) // holds the latest message seen from each node

		for _, e := range entries {
			e.ch <- nil // a nil message means "start a new slot"
			msgs[e.node.ID] = nil
		}

		for looping := true; looping; {
			// After one second of inactivity, resend the latest messages to everyone.
			timer := time.NewTimer(time.Second)

			select {
			case msg := <-ch:
				// Never mind about resending messages.
				if !timer.Stop() {
					<-timer.C
				}
				if msg.I < slotID {
					// discard messages about old slots
					continue
				}
				n := entries[msg.V].node

				if true { // xxx
					if msgs[msg.V] == nil {
						n.Logf("%s", msg)
					} else {
						switch msgs[msg.V].T.(type) {
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
				}
				msgs[msg.V] = msg

				allExt := true
				for _, m := range msgs {
					if m == nil {
						allExt = false
						break
					}
					if _, ok := m.T.(*scp.ExtTopic); !ok {
						allExt = false
						break
					}
				}
				if allExt {
					log.Print("all externalized")
					looping = false
				}

				// Send this message to every other node.
				for nodeID, e := range entries {
					if nodeID == msg.V {
						continue
					}
					e.ch <- msg
				}

			case <-timer.C:
				// It's too quiet around here.
				for nodeID, latest := range msgs {
					if latest == nil {
						continue
					}
					for _, e := range entries {
						n := e.node
						if n.ID == nodeID {
							continue
						}
						err := n.Handle(latest)
						if err != nil {
							n.Logf("could not handle resend of %s: %s", latest, err)
							continue
						}
					}
				}
			}
		}
	}
}

// runs as a goroutine
func nodefn(n *scp.Node, recv <-chan *scp.Msg) {
	for msg := range recv {
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

		err := n.Handle(msg)
		if err != nil {
			n.Logf("could not handle %s: %s", msg, err)
			continue
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

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

	nodes := make(map[scp.NodeID]*scp.Node)

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
		nodes[nodeID] = node
		go node.Run()
	}

	for slotID := scp.SlotID(1); ; slotID++ {
		msgs := make(map[scp.NodeID]*scp.Msg) // holds the latest message seen from each node

		for _, node := range nodes {
			msgs[node.ID] = nil

			// New slot! Nominate something.
			val := foods[rand.Intn(len(foods))]
			nomMsg := scp.NewMsg(node.ID, slotID, node.Q, &scp.NomTopic{X: scp.ValueSet{val}})
			node.Handle(nomMsg)
		}

		for looping := true; looping; {
			// After one second of inactivity, ping each node.
			timer := time.NewTimer(time.Second)

			select {
			case msg := <-ch:
				// Never mind about pinging nodes.
				if !timer.Stop() {
					<-timer.C
				}
				if msg.I < slotID {
					// discard messages about old slots
					continue
				}
				n := nodes[msg.V]

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
				} else {
					n.Logf("%s", msg)
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
				// TODO: every other node with msg.V among its peers.
				for otherNodeID, otherNode := range nodes {
					if otherNodeID == msg.V {
						continue
					}
					otherNode.Handle(msg)
				}

			case <-timer.C:
				// It's too quiet around here.
				for _, node := range nodes {
					node.Ping()
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

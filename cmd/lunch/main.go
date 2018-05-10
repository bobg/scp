package main

// Usage:
//   lunch [-seed N] CONFIGFILE

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"

	"github.com/BurntSushi/toml"
	"github.com/bobg/scp"
)

type valType string

func (v valType) Less(other scp.Value) bool {
	return v < other.(valType)
}

func (v valType) Combine(other scp.Value, slotID scp.SlotID) scp.Value {
	if slotID%2 == 0 {
		if v > other.(valType) {
			return v
		}
	} else if v < other.(valType) {
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

	if flag.NArg() < 1 {
		log.Fatal("usage: lunch [-seed N] CONFFILE")
	}
	confFile := flag.Arg(0)
	confBits, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Fatal(err)
	}
	var conf struct {
		Nodes map[string][][]string
	}
	_, err = toml.Decode(string(confBits), &conf)
	if err != nil {
		log.Fatal(err)
	}

	nodes := make(map[scp.NodeID]*scp.Node)
	ch := make(chan *scp.Msg, 10000)
	for nodeID, qstrs := range conf.Nodes {
		q := make([]scp.NodeIDSet, 0, len(qstrs))
		for _, slice := range qstrs {
			var qslice scp.NodeIDSet
			for _, id := range slice {
				qslice = qslice.Add(scp.NodeID(id))
			}
			q = append(q, qslice)
		}
		node := scp.NewNode(scp.NodeID(nodeID), q, ch, nil)
		nodes[node.ID] = node
		go node.Run(context.Background())
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

		for msg := range ch {
			if msg.I < slotID {
				// discard messages about old slots
				continue
			}
			n := nodes[msg.V]

			if false { // xxx
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
				break
			}

			// Send this message to every other node.
			// TODO: every other node with msg.V among its peers.
			for otherNodeID, otherNode := range nodes {
				if otherNodeID == msg.V {
					continue
				}
				otherNode.Handle(msg)
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

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

func (v valType) IsNil() bool {
	return v == ""
}

func (v valType) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, v)
	return buf.Bytes()
}

func (v valType) String() string {
	return string(v)
}

type nodeconf struct {
	Q  scp.QSet
	FP int
	FQ int
}

func main() {
	seed := flag.Int64("seed", 1, "RNG seed")
	delay := flag.Int("delay", 100, "random delay limit in milliseconds")
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
	var conf map[string]nodeconf
	_, err = toml.Decode(string(confBits), &conf)
	if err != nil {
		log.Fatal(err)
	}

	nodes := make(map[scp.NodeID]*scp.Node)
	ch := make(chan *scp.Msg)
	for nodeID, nconf := range conf {
		node := scp.NewNode(scp.NodeID(nodeID), nconf.Q, ch, nil)
		node.FP, node.FQ = nconf.FP, nconf.FQ
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
			for otherNodeID, otherNode := range nodes {
				if otherNodeID == msg.V {
					continue
				}
				if *delay > 0 {
					otherNode.Delay(rand.Intn(*delay))
				}
				otherNode.Handle(msg)
			}
		}
	}
}

var foods = []valType{
	"burgers",
	"burritos",
	"gyros",
	"indian",
	"pasta",
	"pizza",
	"salads",
	"sandwiches",
	"soup",
	"sushi",
}

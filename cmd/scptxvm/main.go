package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/protocol"
	"github.com/chain/txvm/protocol/bc"

	"github.com/bobg/scp"
)

var (
	chain *protocol.Chain
	node  *scp.Node
	prv   ed25519.PrivateKey
	dir   string

	heightChan = make(chan uint64, 1)
	nomChan    = make(chan interface{}, 1)
	msgChan    = make(chan *scp.Msg, 1)

	msgTimesMu sync.Mutex
	msgTimes   = make(map[scp.NodeID]time.Time)
)

func main() {
	confFile := flag.String("conf", "conf.toml", "config file")
	dirFlag := flag.String("dir", ".", "root of working dir")
	initialBlockFile := flag.String("initial", "", "file containing initial block")

	flag.Parse()

	dir = *dirFlag

	confBits, err := ioutil.ReadFile(*confFile)
	if err != nil {
		log.Fatal(err)
	}

	var conf struct {
		Addr string
		Prv  string
		Q    [][]string
	}
	_, err = toml.Decode(string(confBits), &conf)
	if err != nil {
		log.Fatal(err)
	}

	if *initialBlockFile == "" {
		log.Fatal("must specify -initial")
	}
	initialBlockBits, err := ioutil.ReadFile(*initialBlockFile)
	if err != nil {
		log.Fatal(err)
	}
	var initialBlock bc.Block
	err = initialBlock.FromBytes(initialBlockBits)
	if err != nil {
		log.Fatal(err)
	}

	store := &pstore{
		height:   height,
		snapshot: snapshot,
	}

	heightChan = make(chan uint64)

	chain, err = protocol.NewChain(context.Background(), &initialBlock, store, heightChan)
	if err != nil {
		log.Fatal(err)
	}

	prvBits, err := hex.DecodeString(conf.Prv)
	if err != nil {
		log.Fatal(err)
	}
	if len(prvBits) != ed25519.PrivateKeySize {
		log.Fatalf("prv is %d bytes long, want %d bytes", len(prvBits), ed25519.PrivateKeySize)
	}
	prv = ed25519.PrivateKey(prvBits)
	pubKey := prv.Public().(ed25519.PublicKey)
	pubKeyHex := hex.EncodeToString(pubKey)

	var q []scp.NodeIDSet
	for _, slice := range conf.Q {
		var s scp.NodeIDSet
		for _, id := range slice {
			s = s.Add(id)
		}
		q = append(q, s)
	}

	nodeID := fmt.Sprintf("http://%s/%s", conf.Addr, pubKeyHex)
	node = scp.NewNode(scp.NodeID(nodeID), q, msgChan)
	go node.Run()
	go handleNodeOutput()
	go nominate()
	go subscribe()

	http.HandleFunc("/"+pubKeyHex, protocolHandler) // scp protocol messages go here
	http.HandleFunc("/blocks", blocksHandler)       // nodes resolve block ids here
	http.HandleFunc("/submit", submitHandler)       // new txs get proposed here
	http.HandleFunc("/subscribe", subscribeHandler)
	http.HandleFunc("/shutdown", shutdownHandler)

	node.Logf("node %s listening on %s", node.ID, conf.Addr)
	http.ListenAndServe(conf.Addr, nil)
}

type blocksReq struct {
	Height   int
	BlockIDs []bc.Hash
}

func subscribe() {
	for range time.Tick(time.Minute) {
		// Once per minute, subscribe to other nodes as necessary.
		// "Necessary" is: the other node is in the transitive closure of
		// this node's quorum slices and we have no message from it in the
		// past five minutes.
		others := node.AllKnown()
		for _, other := range others {
			msgTimesMu.Lock()
			t, ok := msgTimes[other]
			msgTimesMu.Unlock()

			if !ok || time.Since(t) > 5*time.Minute {
				u, err := url.Parse(other)
				if err != nil {
					panic(err) // xxx err
				}
				u.Path = "/subscribe"
				u.RawQuery = fmt.Sprintf("subscriber=%s&max=%d", url.QueryEscape(node.ID), highestExt)
				resp, err := http.Get(u.String())
				if err != nil {
					node.Logf("ERROR: cannot subscribe to %s: %s", other, err)
					continue
				}
				defer resp.Body.Close()

				msgTimesMu.Lock()
				msgTimes[other] = time.Now()
				msgTimesMu.Unlock()

				respBits, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					node.Logf("ERROR: reading response: %s", err)
					continue
				}
				var rawMsgs []json.RawMessage
				err = json.Unmarshal(respBits, &rawMsgs)
				if err != nil {
					node.Logf("ERROR: parsing response: %s", err)
					continue
				}
				for _, r := range rawMsgs {
					msg, err := unmarshal(r)
					if err != nil {
						node.Logf("ERROR: parsing protocol message: %s", err)
						continue
					}
					node.Handle(msg)
				}
			}
		}
	}
}

package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"chain/crypto/ed25519"
	"chain/protocol/bc"

	"github.com/bobg/scp"
)

var (
	node *scp.Node

	blockMap   map[string]*bc.Block
	blockMapMu sync.Mutex
)

func main() {
	secretKeyHex := flag.String("seckey", "", "secret key hex")
	addr := flag.String("addr", "", "listen address (host:port)")

	flag.Parse()

	secretKeyBytes, err := hex.DecodeString()
	if err != nil {
		log.Fatal(err)
	}
	if len(secretKey) != ed25519.PrivateKeySize {
		log.Fatalf("secret key is %d bytes long, want %d bytes", len(secretKey), ed25519.PrivateKeySize)
	}
	var (
		secretKey = ed25519.PrivateKey(secretKeyBytes)
		pubKey    = secretKey.Public().(ed25519.PublicKey)
		pubKeyHex = hex.EncodeToString(pubKey)
	)

	// Maps hex-encoded block IDs to the blocks they denote.
	blockMap = make(map[string]*bc.Block)

	nodeID := fmt.Sprintf("http://%s/%x", *addr, pubKey)

	ch := make(chan *scp.Msg)
	node = scp.NewNode(nodeID, q, ch)
	go node.Run()
	go handleNodeOutput(node, ch, secretKey)

	http.HandleFunc("/"+pubKeyHex, inboundHandler)
	http.HandleFunc("/block", blockHandler)

	node.Logf("listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}

func inboundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		// xxx err
	}
	defer r.Body.Close()
	pmsg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// xxx
	}
	msg, err := unmarshal(pmsg)
	if err != nil {
		// xxx
	}
	// xxx parse msg for unknown block ids and request their contents
	node.Handle(msg)
	w.WriteHeader(http.StatusNoContent)
}

func blockHandler(w http.ResponseWriter, r *http.Request) {
	blockIDHex := r.FormValue("id")

	blockMapMu.Lock()
	defer blockMapMu.Unlock()

	block, ok := blockMap[blockIDHex]
	if !ok {
		// xxx err
	}
	blockBytes, err := block.Bytes()
	if err != nil {
		// xxx
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(blockBytes)
	if err != nil {
		// xxx
	}
}

func handleNodeOutput(node *scp.Node, ch <-chan *scp.Msg, seckey ed25519.PrivateKey) {
	var latest *scp.Msg
	ticker := time.Tick(time.Second)

	// xxx also periodically request the latest protocol message from
	// the set of nodes in all quorums to which this node belongs, but
	// which don't have this node in their peer set.

	for {
		select {
		case latest = <-ch:
			// do nothing
		case <-ticker:
			// Send only the latest protocol message (if any) to all peers
			// no more than once per second.
			if latest != nil {
				continue
			}
			var (
				msg  = latest
				pmsg = marshal(msg, seckey)
			)
			latest = nil
			for _, peerID := range node.Peers() {
				peerID := peerID
				go func() {
					resp, err := http.Post(peerID, xxxcontenttype, bytes.NewReader(pmsg))
					if err != nil {
						node.Logf("posting message %s to %s: %s", msg, peerID, err)
						return
					}
					defer resp.Body.Close()
					if resp.StatusCode/100 != 2 {
						node.Logf("unexpected response posting message %s to %s: %s", msg, peerID, resp.Status)
						return
					}
				}()
			}
		}
	}
}

type (
	marshaled struct {
		M marshaledPayload
		S []byte // signature over marshaledPayload
	}

	marshaledPayload struct {
		V string
		I int
		Q [][]string
		T marshaledTopic
	}

	marshaledTopic struct {
		Type        string
		X, Y        []string
		B, C, P, PP marshaledBallot
		HN, CN      int
	}

	marshaledBallot struct {
		N int
		X string
	}
)

func marshal(msg *scp.Msg, prv ed25519.PrivateKey) []byte {
	// xxx
}

func unmarshal(b []byte) (*scp.Msg, error) {
	// xxx
}

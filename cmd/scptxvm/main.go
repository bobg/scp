package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/protocol"
	"github.com/chain/txvm/protocol/bc"
	"github.com/golang/protobuf/proto"
	"golang.org/x/sync/errgroup"

	"github.com/bobg/scp"
)

var (
	chain *protocol.Chain
	node  *scp.Node

	heightChan = make(chan uint64, 1)
	nomChan    = make(chan interface{}, 1)
	msgChan    = make(chan *scp.Msg, 1)
)

func main() {
	secretKeyHex := flag.String("seckey", "", "secret key hex")
	addr := flag.String("addr", "", "listen address (host:port)")
	dir := flag.String("dir", ".", "root of working dir")

	flag.Parse()

	store := &pstore{
		height:   height,
		dir:      *dir,
		snapshot: snapshot,
	}

	heightChan = make(chan uint64)

	var err error

	chain, err = protocol.NewChain(ctx, initialBlock, store, heightChan)
	if err != nil {
		log.Fatal(err)
	}

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

	nodeID := fmt.Sprintf("http://%s/%x", *addr, pubKey)

	node = scp.NewNode(nodeID, q, msgChan)
	go node.Run()
	go handleNodeOutput(node, secretKey)
	go nominate(node)

	http.HandleFunc("/"+pubKeyHex, protocolHandler) // scp protocol messages go here
	http.HandleFunc("/block", blockHandler)         // nodes resolve block ids here
	http.HandleFunc("/submit", submitHandler)       // new txs get proposed here

	node.Logf("listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}

func protocolHandler(w http.ResponseWriter, r *http.Request) {
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

	nh := atomic.LoadInt32(&nomHeight)
	if msg.I >= nh {
		var bump bool
		switch msg.T.(type) {
		case *scp.CommitTopic:
			bump = true
		case *scp.ExtTopic:
			bump = true
		}
		if bump {
			// Can no longer nominate for slot nomHeight.
			atomic.StoreInt32(&nomHeight, msg.I+1)
			nomChan <- msg.I + 1
		}
	}

	var blockIDs scp.ValueSet
	switch topic := msg.T.(type) {
	case *scp.NomTopic:
		blockIDs = blockIDs.Union(topic.X)
		blockIDs = blockIDs.Union(topic.Y)

	case *scp.PrepTopic:
		blockIDs = blockIDs.Add(topic.B.X)
		if !topic.P.IsZero() {
			blockIDs = blockIDs.Add(topic.P.X)
		}
		if !topic.PP.IsZero() {
			blockIDs = blockIDs.Add(topic.PP.X)
		}

	case *scp.CommitTopic:
		blockIDs = blockIDs.Add(topic.B.X)

	case *scp.ExtTopic:
		blockIDs = blockIDs.Add(topic.C.X)
	}

	var c http.Client

	g, ctx := errgroup.WithContext(r.Context())
	for _, blockID := range blockIDs {
		blockID := blockID
		g.Go(func() error {
			if haveBlock(blockID) {
				return nil
			}

			u, err := url.Parse(msg.V)
			if err != nil {
				return err
			}
			u.Path = "/block"
			u.RawQuery = fmt.Sprintf("height=%d&id=%x", msg.I, blockID.(valtype).String())
			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				return err
			}
			req = req.WithContext(r.Context())
			resp, err := c.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			blockBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				// xxx
			}
			var block bc.Block
			err = block.FromBytes(blockBytes)
			if err != nil {
				// xxx
			}

			return storeBlock(block)
		})
	}
	err = g.Wait()
	if err != nil {
		// xxx
	}

	node.Handle(msg)
	w.WriteHeader(http.StatusNoContent)
}

func blockHandler(w http.ResponseWriter, r *http.Request) {
	heightStr := r.FormValue("height")
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		// xxx
	}
	blockIDHex := r.FormValue("id")
	blockID, err := hex.DecodeString(blockIDHex)
	if err != nil {
		// xxx
	}

	block, err := getBlock(height, blockID)
	if err != nil {
		// xxx
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

func submitHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// xxx
	}
	var rawtx bc.RawTx
	err = proto.Unmarshal(b, &tx)
	if err != nil {
		// xxx
	}
	tx, err := bc.NewTx(rawtx.Program, rawtx.Version, rawtx.Runlimit)
	if err != nil {
		// xxx
	}
	nomChan <- tx
	w.WriteHeader(http.StatusNoContent)
}

func handleNodeOutput(node *scp.Node, seckey ed25519.PrivateKey) {
	var latest *scp.Msg
	ticker := time.Tick(time.Second)

	// xxx also periodically request the latest protocol message from
	// the set of nodes in all quorums to which this node belongs, but
	// which don't have this node in their peer set.

	for {
		select {
		case latest = <-msgChan:
			if ext, ok := latest.T.(*scp.ExtTopic); ok {
				// We've externalized a block at a new height.
				// xxx update the tx pool to remove published and conflicting txs

				// Update the protocol.Chain object and anything waiting on it.
				heightChan <- uint64(latest.I)
			}

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
		M json.RawMessage
		S []byte // signature over marshaledPayload
	}

	marshaledPayload struct {
		C int
		V string
		I int
		Q [][]string
		T marshaledTopic
	}

	marshaledTopic struct {
		Type        int // scp.Phase values
		X, Y        []string
		B, C, P, PP marshaledBallot
		PN, HN, CN  int
	}

	marshaledBallot struct {
		N int
		X string
	}
)

func marshal(msg *scp.Msg, prv ed25519.PrivateKey) ([]byte, error) {
	var q [][]string
	for _, slice := range msg.Q {
		var qslice []string
		for _, id := range slice {
			qslice = append(qslice, id)
		}
		q = append(q, qslice)
	}

	var mt marshaledTopic
	switch topic := msg.T.(type) {
	case *scp.NomTopic:
		// xxx build x and y
		mt.X = x
		mt.Y = y

	case *scp.PrepTopic:
		mt.B = marshaledBallot{N: topic.B.N, X: topic.B.X.(valtype).String()}
		mt.P = marshaledBallot{N: topic.P.N, X: topic.P.X.(valtype).String()}
		mt.PP = marshaledBallot{N: topic.PP.N, X: topic.PP.X.(valtype).String()}
		mt.HN = topic.HN
		mt.CN = topic.CN

	case *scp.CommitTopic:
		mt.B = marshaledBallot{N: topic.B.N, X: topic.B.X.(valtype).String()}
		mt.PN = topic.PN
		mt.HN = topic.HN
		mt.CN = topic.CN

	case *scp.ExtTopic:
		mt.C = marshaledBallot{N: topic.C.N, topic.C.X.(valtype).String()}
		mt.HN = topic.HN
	}
	mp := marshaledPayload{
		C: msg.C,
		V: msg.V,
		I: msg.I,
		Q: q,
		T: mt,
	}
	mpbytes, err := json.Marshal(mp) // xxx json is subject to mutation in transit!
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(prv, mpbytes)
	m := marshaled{
		M: mpbytes,
		S: sig,
	}
	return json.Marshal(m)
}

func unmarshal(b []byte) (*scp.Msg, error) {
	var m marshaled
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(pubkey, m.M, m.S) {
		return nil, errors.New("bad signature")
	}

	var mp marshaledPayload
	err = json.Unmarshal(m.M, &mp)
	if err != nil {
		return nil, err
	}

	var q []scp.NodeIDSet
	for _, slice := range mp.Q {
		var qslice scp.NodeIDSet
		for _, id := range slice {
			qslice = qslice.Add(id)
		}
		q = append(q, qslice)
	}

	var topic scp.Topic
	switch mp.T.Type {
	case scp.PhNom:
		topic = &scp.NomTopic{
			X: x,
			Y: y,
		}

	case scp.PhPrep:
		topic = &scp.PrepTopic{
			B:  b,
			P:  p,
			PP: pp,
			HN: mp.T.HN,
			CN: mp.T.CN,
		}

	case scp.PhCommit:
		topic = &scp.CommitTopic{
			B:  b,
			PN: mp.T.PN,
			HN: mp.T.HN,
			CN: mp.T.CN,
		}

	case scp.PhExt:
		topic = &scp.ExtTopic{
			C:  c,
			HN: mp.T.HN,
		}

	default:
		return nil, fmt.Errorf("unknown topic type %d", mp.T.Type)
	}

	msg := &scp.Msg{
		C: mp.C,
		V: mp.V,
		I: mp.I,
		Q: q,
		T: topic,
	}
	return msg, nil
}

func nominate(node *scp.Node) {
	txpool := make(map[bc.Hash]*bc.Tx)

	doNom := func() error {
		txs := make([]*bc.Tx, 0, len(txpool))
		for _, tx := range txpool {
			txs = append(txs, tx)
		}

		block, _, err := chain.GenerateBlock(ctx, chain.State(), timestampMS, txs)
		if err != nil {
			return err
		}
		// xxx figure out which txs GenerateBlock removed as invalid, and remove them from txpool

		err = storeBlock(block)
		if err != nil {
			return err
		}
		n := &scp.NomTopic{
			X: scp.ValueSet{block.Hash()},
		}
		msg := scp.NewMsg(node.ID, scp.SlotID(block.Height), node.Q, n) // xxx slotID is 32 bits, block height is 64
		node.Handle(msg)
	}

	for item := range nomChan {
		switch item := item.(type) {
		case *bc.Tx:
			if _, ok := txpool[item.ID]; ok {
				// tx is already in the pool
				continue
			}
			txpool[item.ID] = item // xxx need to persist this
			err := doNom()
			if err != nil {
				// xxx
			}

		case scp.SlotID:
			err := doNom()
			if err != nil {
				// xxx
			}
		}
	}
}

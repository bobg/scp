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
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/protocol"
	"github.com/chain/txvm/protocol/bc"
	"github.com/golang/protobuf/proto"

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

	chain, err = protocol.NewChain(ctx, &initialBlock, store, heightChan)
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

	nodeID := fmt.Sprintf("http://%s/%s", conf.Addr, pubKeyHex)

	var q []scp.NodeIDSet
	for _, slice := range conf.Q {
		var s scp.NodeIDSet
		for _, id := range slice {
			s = s.Add(id)
		}
		q = append(q, s)
	}

	node = scp.NewNode(nodeID, q, msgChan)
	go node.Run()
	go handleNodeOutput()
	go nominate()

	http.HandleFunc("/"+pubKeyHex, protocolHandler) // scp protocol messages go here
	http.HandleFunc("/blocks", blocksHandler)       // nodes resolve block ids here
	http.HandleFunc("/submit", submitHandler)       // new txs get proposed here
	http.HandleFunc("/shutdown", shutdownHandler)

	node.Logf("listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}

type blocksReq struct {
	Height   int
	BlockIDs []bc.Hash
}

func protocolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		httperr(w, http.StatusBadRequest, "%s not supported", r.Method)
		return
	}
	if r.Body == nil {
		httperr(w, http.StatusBadRequest, "missing POST body")
		return
	}
	defer r.Body.Close()
	pmsg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httperr(w, http.StatusInternalServerError, "could not read POST body: %s", err)
		return
	}
	msg, err := unmarshal(pmsg)
	if err != nil {
		httperr(w, http.StatusBadRequest, "could not parse POST body: %s", err)
		return
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

	// Collect all block IDs mentioned in the new message.
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

	// Request the contents of any unknown blocks.
	req := blocksReq{Height: msg.I}
	for _, blockID := range blockIDs {
		blockID := bc.Hash(blockID.(valtype))
		have, err := haveBlock(msg.I, blockID)
		if err != nil {
			httperr(w, http.StatusInternalServerError, "could not check for block file: %s", err)
			return
		}
		if have {
			continue
		}
		req.BlockIDs = append(req.BlockIDs, blockID)
	}
	if len(req.Blocks) > 0 {
		u, err := url.Parse(msg.V)
		if err != nil {
			httperr(w, http.StatusBadRequest, "sending node ID (%s) cannot be parsed as a URL: %s", msg.V, err)
			return
		}
		u.Path = "/blocks"
		body, err := json.Marshal(req)
		if err != nil {
			httperr(w, http.StatusInternalServerError, "cannot construct POST body: %s", err)
			return
		}
		req, err := http.NewRequest("POST", u.String(), bytes.NewReader(body))
		if err != nil {
			httperr(w, http.StatusInternalServerError, "building POST request: %s", err)
			return
		}
		req = req.WithContext(r.Context())
		req.Header.Set("Content-Type", "application/json")
		var c http.Client
		resp, err := c.Do(req)
		if err != nil {
			httperr(w, http.StatusInternalServerError, "requesting block contents: %s", err)
			return
		}
		// xxx check status code and content-type
		defer resp.Body.Close()
		respBits, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			httperr(w, http.StatusInternalServerError, "reading response: %s", err)
			return
		}
		var blocks []*bc.Block
		err = json.Unmarshal(respBits, &blocks)
		if err != nil {
			httperr(w, http.StatusInternalServerError, "parsing response: %s", err)
			return
		}
		// xxx check all requested blocks are present
		for _, block = range blocks {
			err = storeBlock(block)
			if err != nil {
				httperr(w, http.StatusInternalServerError, "storing block: %s", err)
				return
			}
		}
	}

	node.Handle(msg)
	w.WriteHeader(http.StatusNoContent)
}

func blocksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		httperr(w, http.StatusBadRequest, "%s not supported", r.Method)
		return
	}
	if r.Body == nil {
		httperr(w, http.StatusBadRequest, "missing POST body")
		return
	}
	defer r.Body.Close()
	reqBits, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httperr(w, http.StatusInternalServerError, "reading request: %s", err)
		return
	}
	var req blocksReq
	err = json.Unmarshal(reqBits, &req)
	if err != nil {
		httperr(w, http.StatusBadRequest, "parsing request: %s", err)
		return
	}
	var result []*bc.Block
	for _, blockID := range req.BlockIDs {
		block, err := getBlock(req.Height, blockID)
		if err != nil {
			httperr(w, http.StatusNotFound, "could not resolve requested block %s (height %d): %s", blockID, req.Height, err)
			return
		}
		result = append(result, block)
	}
	respBits, err := json.Marshal(result)
	if err != nil {
		httperr(w, http.StatusInternalServerError, "could not marshal response: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respBits)
	if err != nil {
		httperr(w, http.StatusInternalServerError, "could not write response: %s", err)
		return
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		httperr(w, http.StatusBadRequest, "%s not supported", r.Method)
		return
	}
	if r.Body == nil {
		httperr(w, http.StatusBadRequest, "missing POST body")
		return
	}
	defer r.Body.Close()
	reqBits, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httperr(w, http.StatusInternalServerError, "reading request: %s", err)
		return
	}
	var rawtx bc.RawTx
	err = proto.Unmarshal(reqBits, &tx)
	if err != nil {
		httperr(w, http.StatusBadRequest, "parsing request: %s", err)
		return
	}
	tx, err := bc.NewTx(rawtx.Program, rawtx.Version, rawtx.Runlimit)
	if err != nil {
		httperr(w, http.StatusBadRequest, "validating transaction: %s", err)
		return
	}
	nomChan <- tx
	w.WriteHeader(http.StatusNoContent)
}

func handleNodeOutput() {
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
				pmsg = marshal(msg)
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

func marshal(msg *scp.Msg) ([]byte, error) {
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

func nominate() {
	txpool := make(map[bc.Hash]*bc.Tx)

	doNom := func() error {
		if len(txpool) == 0 {
			return nil
		}

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
				panic(err) // xxx
			}

		case scp.SlotID:
			err := doNom()
			if err != nil {
				panic(err) // xxx
			}
		}
	}
}

func httperr(w http.ResponseWriter, code int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	node.Logf("http response %d: %s", code, msg)
	http.Error(w, msg, code)
}

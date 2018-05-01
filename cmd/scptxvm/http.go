package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/bobg/scp"

	"github.com/chain/txvm/protocol/bc"
)

var nomHeight int32

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

	msgTimesMu.Lock()
	msgTimes[msg.V] = time.Now()
	msgTimesMu.Unlock()

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

func httperr(w http.ResponseWriter, code int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	node.Logf("http response %d: %s", code, msg)
	http.Error(w, msg, code)
}

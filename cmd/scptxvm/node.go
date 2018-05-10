package main

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/bobg/scp"
	"github.com/chain/txvm/protocol/bc"
)

var (
	// Value is when the subscriber subscribed.
	subscribers   = make(map[scp.NodeID]time.Time)
	subscribersMu sync.Mutex
)

func handleNodeOutput(ctx context.Context) {
	defer wg.Done()

	var latest *scp.Msg
	ticker := time.Tick(time.Second)

	for {
		select {
		case <-ctx.Done():
			node.Logf("context canceled, exiting node-output loop")
			return

		case latest = <-msgChan:
			if ext, ok := latest.T.(*scp.ExtTopic); ok {
				// We've externalized a block at a new height.

				// Update the tx pool to remove published and conflicting txs.
				block, err := getBlock(int(latest.I), bc.Hash(ext.C.X.(valtype)))
				if err != nil {
					panic(err) // xxx
				}
				nomChan <- block

				// Update the protocol.Chain object and anything waiting on it.
				heightChan <- uint64(latest.I)
			}

		case <-ticker:
			// Send only the latest protocol message (if any) to all peers
			// and subscribers no more than once per second.
			if latest == nil {
				continue
			}
			msg := latest
			pmsg, err := marshal(msg)
			if err != nil {
				panic(err) // xxx
			}

			latest = nil

			others := node.Peers()
			subscribersMu.Lock()
			for other := range subscribers {
				others = others.Add(other)
			}
			subscribersMu.Unlock()

			for _, other := range others {
				other := other
				go func() {
					node.Logf("* sending %s to %s", msg, other)
					req, err := http.NewRequest("POST", string(other), bytes.NewReader(pmsg))
					if err != nil {
						node.Logf("error constructing protocol request to %s: %s", other, err)
						return
					}
					req = req.WithContext(ctx)
					req.Header.Set("Content-Type", "application/octet-stream")
					var c http.Client
					resp, err := c.Do(req)
					if err != nil {
						node.Logf("could not send protocol message to %s: %s", other, err)
						subscribersMu.Lock()
						delete(subscribers, other)
						subscribersMu.Unlock()
					} else if resp.StatusCode/100 != 2 {
						node.Logf("unexpected status code %d sending protocol message to %s", resp.StatusCode, other)
						subscribersMu.Lock()
						delete(subscribers, other)
						subscribersMu.Unlock()
					}
				}()
			}
		}
	}
}

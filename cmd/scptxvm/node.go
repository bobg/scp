package main

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/bobg/scp"
)

var (
	// Value is when the subscriber subscribed.
	subscribers   = make(map[scp.NodeID]time.Time)
	subscribersMu sync.Mutex
)

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
			// and subscribers no more than once per second.
			if latest != nil {
				continue
			}
			msg := latest
			pmsg := marshal(msg)
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
					resp, err := http.Post(other, "application/octet-stream", bytes.NewReader(pmsg))
					if err != nil {
						node.Logf("could not send protocol message to %s: %s", other, err)
						subscribersMu.Lock()
						delete(subscribers, other)
						subscribersMu.Unlock()
					}
					if resp.StatusCode/100 != 2 {
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

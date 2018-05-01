package main

import (
	"bytes"
	"net/http"
	"time"

	"github.com/bobg/scp"
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

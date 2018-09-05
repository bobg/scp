package main

import (
	"context"
	"sync"
	"time"

	"github.com/bobg/scp"
)

func runNode(ctx context.Context, wg *sync.WaitGroup, node *scp.Node) {
	defer wg.Done()

	node.Run(ctx)
}

func handleNodeOutput(ctx context.Context, wg *sync.WaitGroup, msgs <-chan *scp.Msg) {
	defer wg.Done()

	var latest *scp.Msg
	ticker := time.Tick(time.Second)

	for {
		select {
		case <-ctx.Done():
			return

		case latest = <-msgs:
			if ext := latest.T.ExtTopic; ext != nil {
				// We've externalized a new value.
				// xxx
			}

		case <-ticker:
			// Multicast the latest protocol message (if any).
			// xxx
		}
	}
}

func nominate(ctx context.Context, wg *sync.WaitGroup, node *scp.Node) {
	defer wg.Done()

}

func udpListener(ctx context.Context, wg *sync.WaitGroup, node *scp.Node) {
	defer wg.Done()

}

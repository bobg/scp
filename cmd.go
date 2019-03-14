package scp

import (
	"context"
	"sync"
)

// Commands for the Node goroutine.

type Cmd interface{}

type msgCmd struct {
	msg *Msg
}

type deferredUpdateCmd struct {
	slot *Slot
}

type newRoundCmd struct {
	slot *Slot
}

type rehandleCmd struct {
	slot *Slot
}

type delayCmd struct {
	ms int
}

// Internal channel for queueing and processing commands.

type cmdChan struct {
	cond sync.Cond
	cmds []Cmd
}

func newCmdChan() *cmdChan {
	result := new(cmdChan)
	result.cond.L = new(sync.Mutex)
	return result
}

func (c *cmdChan) write(cmd Cmd) {
	c.cond.L.Lock()
	c.cmds = append(c.cmds, cmd)
	c.cond.Broadcast()
	c.cond.L.Unlock()
}

func (c *cmdChan) read(ctx context.Context) (Cmd, bool) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for len(c.cmds) == 0 {
		ch := make(chan struct{})
		go func() {
			c.cond.Wait()
			close(ch)
		}()

		select {
		case <-ctx.Done():
			return nil, false

		case <-ch:
			if len(c.cmds) == 0 {
				continue
			}
		}
	}
	result := c.cmds[0]
	c.cmds = c.cmds[1:]
	return result, true
}

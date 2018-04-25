package scp

// Commands for the Node goroutine.

type Cmd interface{}

type msgCmd struct {
	msg *Msg
}

type deferredUpdateCmd struct {
	slotID SlotID
}

type pingCmd struct{}

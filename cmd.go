package scp

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

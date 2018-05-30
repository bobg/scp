package scp

// Commands for the Node goroutine.

type Cmd interface{}

type msgCmd struct {
	msg *Msg
}

type deferredUpdateCmd struct {
	slotID SlotID
}

type newRoundCmd struct {
	slot *Slot
}

type rehandleCmd struct {
	slot *Slot
}

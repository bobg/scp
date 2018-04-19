package scp

// Env is the envelope of an SCP protocol message.
type Env struct {
	V NodeID
	I SlotID
	Q *QSet
	M Msg
}

package scp

// Env is the envelope of an SCP protocol message.
type Env struct {
	V NodeID
	I SlotID
	Q *QSet
	M Msg
}

func (e *Env) Less(other *Env) bool {
	// xxx do we need to compare e.V, e.I, e.Q? don't think so
	return e.M.Less(other.M)
}

package scp

import (
	"fmt"
	"sync/atomic"
)

// Env is the envelope of an SCP protocol message.
type Env struct {
	C int32      // A counter for identifying this envelope, does not participate in the protocol.
	V NodeID     // ID of the node sending this message.
	I SlotID     // ID of the slot that this message is about.
	Q [][]NodeID // Quorum slices of the sending node.
	M Msg        // The payload: a *NomMsg, *PrepMsg, *CommitMsg, or *ExtMsg.
}

var msgCounter int32

// NewEnv produces a new envelope.
func NewEnv(v NodeID, i SlotID, q [][]NodeID, m Msg) *Env {
	c := atomic.AddInt32(&msgCounter, 1)
	return &Env{
		C: c,
		V: v,
		I: i,
		Q: q,
		M: m,
	}
}

// Tells whether e votes nominate(v) or accepts nominate(v).
func (e *Env) votesOrAcceptsNominated(v Value) bool {
	if e.acceptsNominated(v) {
		return true
	}
	msg, ok := e.M.(*NomMsg)
	return ok && msg.X.Contains(v)
}

// Tells whether e accepts nominate(v).
func (e *Env) acceptsNominated(v Value) bool {
	switch msg := e.M.(type) {
	case *NomMsg:
		return msg.Y.Contains(v)

	case *PrepMsg:
		return VEqual(msg.B.X, v) || VEqual(msg.P.X, v) || VEqual(msg.PP.X, v)

	case *CommitMsg:
		return VEqual(msg.B.X, v)

	case *ExtMsg:
		return VEqual(msg.C.X, v)
	}
	return false // not reached
}

// Tells whether e votes prepared(b) or accepts prepared(b).
func (e *Env) votesOrAcceptsPrepared(b Ballot) bool {
	if e.acceptsPrepared(b) {
		return true
	}
	msg, ok := e.M.(*PrepMsg)
	return ok && b.Equal(msg.B)
}

// Tells whether e accepts prepared(b).
func (e *Env) acceptsPrepared(b Ballot) bool {
	switch msg := e.M.(type) {
	case *PrepMsg:
		if b.Equal(msg.P) || b.Equal(msg.PP) {
			return true
		}
		if msg.HN > 0 {
			if b.Equal(Ballot{N: msg.HN, X: msg.B.X}) {
				return true
			}
			if msg.CN > 0 {
				// include "vote commit" as "accept prepared"
				return msg.CN <= b.N && b.N <= msg.HN && VEqual(b.X, msg.B.X)
			}
		}

	case *CommitMsg:
		if VEqual(b.X, msg.B.X) {
			// include "vote commit" and "accept commit" as "accept prepared"
			return b.N >= msg.CN || b.N == msg.PN
		}

	case *ExtMsg:
		if VEqual(b.X, msg.C.X) {
			return b.N >= msg.C.N
		}
	}
	return false
}

// Tells whether e votes commit(b) or accepts commit(b) for any ballot
// b whose value is v and whose counter is in the range [min,max]
// (inclusive). If so, returns the new min/max that is the overlap
// between the input and what e votes for or accepts.
func (e *Env) votesOrAcceptsCommit(v Value, min, max int) (bool, int, int) {
	if res, newMin, newMax := e.acceptsCommit(v, min, max); res {
		// xxx newMin/newMax might be too narrow after accounting only for
		// "accepts" and not yet for "votes." do we care?
		return true, newMin, newMax
	}
	switch msg := e.M.(type) {
	case *PrepMsg:
		if msg.CN == 0 || !VEqual(msg.B.X, v) {
			return false, 0, 0
		}
		if msg.CN > max || msg.HN < min {
			return false, 0, 0
		}
		if msg.CN > min {
			min = msg.CN
		}
		if msg.HN < max {
			max = msg.HN
		}
		return true, min, max

	case *CommitMsg:
		if !VEqual(msg.B.X, v) {
			return false, 0, 0
		}
		if msg.CN > max {
			return false, 0, 0
		}
		if msg.CN > min {
			min = msg.CN
		}
		return true, min, max
	}
	return false, 0, 0
}

// Tells whether e accepts commit(b) for any ballot b whose value is v
// and whose counter is in the range [min,max] (inclusive). If so,
// returns the new min/max that is the overlap between the input and
// what e accepts.
func (e *Env) acceptsCommit(v Value, min, max int) (bool, int, int) {
	switch msg := e.M.(type) {
	case *CommitMsg:
		if !VEqual(msg.B.X, v) {
			return false, 0, 0
		}
		if msg.CN > max {
			return false, 0, 0
		}
		if msg.HN < min {
			return false, 0, 0
		}
		if msg.CN > min {
			min = msg.CN
		}
		if msg.HN < max {
			max = msg.HN
		}
		return true, min, max

	case *ExtMsg:
		if !VEqual(msg.C.X, v) {
			return false, 0, 0
		}
		if msg.C.N > max {
			return false, 0, 0
		}
		if msg.C.N > min {
			min = msg.C.N
		}
		return true, min, max
	}
	return false, 0, 0
}

// String produces a readable representation of an envelope.
func (e *Env) String() string {
	return fmt.Sprintf("(C=%d V=%s I=%d: %s)", e.C, e.V, e.I, e.M)
}

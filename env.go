package scp

import (
	"fmt"
	"sync/atomic"
)

// Env is the envelope of an SCP protocol message.
type Env struct {
	C int32
	V NodeID
	I SlotID
	Q [][]NodeID
	M Msg
}

var msgCounter int32

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

func (e *Env) votesOrAcceptsNominated(v Value) bool {
	if e.acceptsNominated(v) {
		return true
	}
	msg, ok := e.M.(*NomMsg)
	return ok && msg.X.Contains(v)
}

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

// Tells whether this message votes for or accepts as prepared the
// given ballot.
func (e *Env) votesOrAcceptsPrepared(b Ballot) bool {
	if e.acceptsPrepared(b) {
		return true
	}
	msg, ok := e.M.(*PrepMsg)
	return ok && b.Equal(msg.B)
}

// Tells whether this message accepts as prepared the given ballot.
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

// Tells whether e accepts as committed any ballots with the given
// value and counter in the given range. If it does, it returns the
// min/max range of such ballots it does accept (i.e., the overlap
// with the input min/max).
func (e *Env) votesOrAcceptsCommit(v Value, min, max int) (bool, int, int) {
	if res, newMin, newMax := e.acceptsCommit(v, min, max); res {
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

// Tells whether e accepts as committed any ballots with the given
// value and counter in the given range. If it does, it returns the
// min/max range of such ballots it does accept (i.e., the overlap
// with the input min/max).
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

func (e *Env) String() string {
	return fmt.Sprintf("(C=%d V=%s I=%d: %s)", e.C, e.V, e.I, e.M)
}

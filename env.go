package scp

import (
	"fmt"
	"math"
)

// Env is the envelope of an SCP protocol message.
type Env struct {
	V NodeID
	I SlotID
	Q [][]NodeID
	M Msg
}

// Tells whether this message votes for or accepts as prepared the
// given ballot.
func (e *Env) votesOrAcceptsPrepared(b Ballot) bool {
	switch msg := e.M.(type) {
	case *PrepMsg:
		if b.Equal(msg.B) || b.Equal(msg.P) || b.Equal(msg.PP) {
			return true
		}
		return b.Equal(Ballot{N: msg.HN, X: msg.B.X})

	case *CommitMsg:
		if b.Equal(Ballot{N: math.MaxInt32, X: msg.B.X}) {
			return true
		}
		if b.Equal(Ballot{N: msg.PN, X: msg.B.X}) {
			return true
		}
		return b.Equal(Ballot{N: msg.HN, X: msg.B.X})

	case *ExtMsg:
		return b.Equal(Ballot{N: math.MaxInt32, X: msg.C.X})
	}

	return false
}

// Tells whether this message accepts as prepared the given ballot.
func (e *Env) acceptsPrepared(b Ballot) bool {
	switch msg := e.M.(type) {
	case *PrepMsg:
		if b.Equal(msg.P) || b.Equal(msg.PP) {
			return true
		}
		return b.Equal(Ballot{N: msg.HN, X: msg.B.X})

	case *CommitMsg:
		if b.Equal(Ballot{N: msg.PN, X: msg.B.X}) {
			return true
		}
		return b.Equal(Ballot{N: msg.HN, X: msg.B.X})

	case *ExtMsg:
		return b.Equal(Ballot{N: math.MaxInt32, X: msg.C.X})
	}

	return false
}

func (e *Env) String() string {
	return fmt.Sprintf("(V=%s, I=%d: %s)", e.V, e.I, e.M)
}

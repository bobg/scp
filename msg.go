package scp

import (
	"fmt"
	"sync/atomic"
)

// Msg is an SCP protocol message.
type Msg struct {
	C int32       // A counter for identifying this envelope, does not participate in the protocol.
	V NodeID      // ID of the node sending this message.
	I SlotID      // ID of the slot that this message is about.
	Q []NodeIDSet // Quorum slices of the sending node.
	T Topic       // The payload: a *NomTopic, *PrepTopic, *CommitTopic, or *ExtTopic.
}

var msgCounter int32

// NewMsg produces a new message.
func NewMsg(v NodeID, i SlotID, q []NodeIDSet, t Topic) *Msg {
	c := atomic.AddInt32(&msgCounter, 1)
	return &Msg{
		C: c,
		V: v,
		I: i,
		Q: q,
		T: t,
	}
}

// Returns the set of values that e votes or accepts as nominated.
func (e *Msg) votesOrAcceptsNominatedSet() ValueSet {
	result := e.acceptsNominatedSet()
	if topic, ok := e.T.(*NomTopic); ok {
		result = result.Union(topic.X)
	}
	return result
}

// Returns the set of values that e accepts as nominated.
func (e *Msg) acceptsNominatedSet() ValueSet {
	switch topic := e.T.(type) {
	case *NomTopic:
		return topic.Y

	case *PrepTopic:
		var s ValueSet
		s = s.Add(topic.B.X)
		if !topic.P.IsZero() {
			s = s.Add(topic.P.X)
		}
		if !topic.PP.IsZero() {
			s = s.Add(topic.PP.X)
		}
		return s

	case *CommitTopic:
		return ValueSet{topic.B.X}

	case *ExtTopic:
		return ValueSet{topic.C.X}
	}
	return nil // not reached
}

// Tells whether e votes prepared(b) or accepts prepared(b).
func (e *Msg) votesOrAcceptsPrepared(b Ballot) bool {
	if e.acceptsPrepared(b) {
		return true
	}
	topic, ok := e.T.(*PrepTopic)
	return ok && b.Equal(topic.B)
}

// Tells whether e accepts prepared(b).
func (e *Msg) acceptsPrepared(b Ballot) bool {
	switch topic := e.T.(type) {
	case *PrepTopic:
		if b.Equal(topic.P) || b.Equal(topic.PP) {
			return true
		}
		if topic.HN > 0 {
			if b.Equal(Ballot{N: topic.HN, X: topic.B.X}) {
				return true
			}
			if topic.CN > 0 {
				// include "vote commit" as "accept prepared"
				return topic.CN <= b.N && b.N <= topic.HN && ValueEqual(b.X, topic.B.X)
			}
		}

	case *CommitTopic:
		if ValueEqual(b.X, topic.B.X) {
			// include "vote commit" and "accept commit" as "accept prepared"
			return b.N >= topic.CN || b.N == topic.PN
		}

	case *ExtTopic:
		if ValueEqual(b.X, topic.C.X) {
			return b.N >= topic.C.N
		}
	}
	return false
}

// Tells whether e votes commit(b) or accepts commit(b) for any ballot
// b whose value is v and whose counter is in the range [min,max]
// (inclusive). If so, returns the new min/max that is the overlap
// between the input and what e votes for or accepts.
func (e *Msg) votesOrAcceptsCommit(v Value, min, max int) (bool, int, int) {
	if res, newMin, newMax := e.acceptsCommit(v, min, max); res {
		// xxx newMin/newMax might be too narrow after accounting only for
		// "accepts" and not yet for "votes." do we care?
		return true, newMin, newMax
	}
	switch topic := e.T.(type) {
	case *PrepTopic:
		if topic.CN == 0 || !ValueEqual(topic.B.X, v) {
			return false, 0, 0
		}
		if topic.CN > max || topic.HN < min {
			return false, 0, 0
		}
		if topic.CN > min {
			min = topic.CN
		}
		if topic.HN < max {
			max = topic.HN
		}
		return true, min, max

	case *CommitTopic:
		if !ValueEqual(topic.B.X, v) {
			return false, 0, 0
		}
		if topic.CN > max {
			return false, 0, 0
		}
		if topic.CN > min {
			min = topic.CN
		}
		return true, min, max
	}
	return false, 0, 0
}

// Tells whether e accepts commit(b) for any ballot b whose value is v
// and whose counter is in the range [min,max] (inclusive). If so,
// returns the new min/max that is the overlap between the input and
// what e accepts.
func (e *Msg) acceptsCommit(v Value, min, max int) (bool, int, int) {
	switch topic := e.T.(type) {
	case *CommitTopic:
		if !ValueEqual(topic.B.X, v) {
			return false, 0, 0
		}
		if topic.CN > max {
			return false, 0, 0
		}
		if topic.HN < min {
			return false, 0, 0
		}
		if topic.CN > min {
			min = topic.CN
		}
		if topic.HN < max {
			max = topic.HN
		}
		return true, min, max

	case *ExtTopic:
		if !ValueEqual(topic.C.X, v) {
			return false, 0, 0
		}
		if topic.C.N > max {
			return false, 0, 0
		}
		if topic.C.N > min {
			min = topic.C.N
		}
		return true, min, max
	}
	return false, 0, 0
}

// String produces a readable representation of a message.
func (e *Msg) String() string {
	return fmt.Sprintf("(C=%d V=%s I=%d: %s)", e.C, e.V, e.I, e.T)
}

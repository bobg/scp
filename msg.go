package scp

import (
	"errors"
	"fmt"
	"math"
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

func (e *Msg) valid() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: %s", err, e)
		}
	}()

	switch topic := e.T.(type) {
	case *NomTopic:
		if len(topic.X.Intersection(topic.Y)) != 0 {
			return errors.New("non-empty intersection between X and Y")
		}

	case *PrepTopic:
		if !topic.P.IsZero() {
			if topic.B.Less(topic.P) {
				return errors.New("P > B")
			}
			if !topic.PP.IsZero() && !topic.PP.Less(topic.P) {
				return errors.New("PP >= P")
			}
		}
		if topic.CN > topic.HN {
			return errors.New("CN > HN (prepare)")
		}
		if topic.HN > topic.B.N {
			return errors.New("HN > BN")
		}

	case *CommitTopic:
		if topic.CN > topic.HN {
			return errors.New("CN > HN (commit)")
		}
	}
	return nil
}

// Return the ballot counter (if any).
func (e *Msg) bN() int {
	switch topic := e.T.(type) {
	case *PrepTopic:
		return topic.B.N

	case *CommitTopic:
		return topic.B.N

	case *ExtTopic:
		return math.MaxInt32
	}
	return 0
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

// Returns the set of ballots for which e votes or accepts "prepared."
func (e *Msg) votesOrAcceptsPreparedSet() BallotSet {
	result := e.acceptsPreparedSet()
	switch topic := e.T.(type) {
	case *PrepTopic:
		result = result.Add(topic.B)

	case *CommitTopic:
		result = result.Add(Ballot{N: math.MaxInt32, X: topic.B.X})
	}
	return result
}

// Returns the set of ballots that e accepts as prepared.
func (e *Msg) acceptsPreparedSet() BallotSet {
	var result BallotSet
	switch topic := e.T.(type) {
	case *PrepTopic:
		if !topic.P.IsZero() {
			result = result.Add(topic.P)
			if !topic.PP.IsZero() {
				result = result.Add(topic.PP)
			}
		}
		if topic.HN > 0 {
			result = result.Add(Ballot{N: topic.HN, X: topic.B.X})
		}

	case *CommitTopic:
		result = result.Add(Ballot{N: topic.PN, X: topic.B.X})
		result = result.Add(Ballot{N: topic.HN, X: topic.B.X})

	case *ExtTopic:
		result = result.Add(Ballot{N: math.MaxInt32, X: topic.C.X})
	}
	return result
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

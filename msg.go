package scp

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"
)

// Msg is an SCP protocol message.
type Msg struct {
	C int32  // A counter for identifying this envelope, does not participate in the protocol.
	V NodeID // ID of the node sending this message.
	I SlotID // ID of the slot that this message is about.
	Q QSet   // Quorum slices of the sending node.
	T Topic  // The payload: a *NomTopic, a *NomPrepTopic, *PrepTopic, *CommitTopic, or *ExtTopic.
}

var msgCounter int32

// NewMsg produces a new message.
func NewMsg(v NodeID, i SlotID, q QSet, t Topic) *Msg {
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

	nom := func(topic *NomTopic) error {
		if len(topic.X.Intersection(topic.Y)) != 0 {
			return errors.New("non-empty intersection between X and Y")
		}
		return nil
	}
	prep := func(topic *PrepTopic) error {
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
		return nil
	}

	switch topic := e.T.(type) {
	case *NomTopic:
		err := nom(topic)
		if err != nil {
			return err
		}

	case *NomPrepTopic:
		err := nom(&topic.NomTopic)
		if err != nil {
			return err
		}
		return prep(&topic.PrepTopic)

	case *PrepTopic:
		return prep(topic)

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
	case *NomPrepTopic:
		return topic.B.N

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
	f := func(topic *NomTopic) {
		result = result.Union(topic.X)
	}
	switch topic := e.T.(type) {
	case *NomTopic:
		f(topic)
	case *NomPrepTopic:
		f(&topic.NomTopic)
	}
	return result
}

// Returns the set of values that e accepts as nominated.
func (e *Msg) acceptsNominatedSet() ValueSet {
	switch topic := e.T.(type) {
	case *NomTopic:
		return topic.Y

	case *NomPrepTopic:
		return topic.Y
	}
	return nil
}

// Returns the set of ballots for which e votes or accepts "prepared."
func (e *Msg) votesOrAcceptsPreparedSet() BallotSet {
	result := e.acceptsPreparedSet()
	f := func(topic *PrepTopic) {
		result = result.Add(topic.B)
	}

	switch topic := e.T.(type) {
	case *NomPrepTopic:
		f(&topic.PrepTopic)

	case *PrepTopic:
		f(topic)

	case *CommitTopic:
		result = result.Add(Ballot{N: math.MaxInt32, X: topic.B.X})
	}
	return result
}

// Returns the set of ballots that e accepts as prepared.
func (e *Msg) acceptsPreparedSet() BallotSet {
	var result BallotSet
	f := func(topic *PrepTopic) {
		if !topic.P.IsZero() {
			result = result.Add(topic.P)
			if !topic.PP.IsZero() {
				result = result.Add(topic.PP)
			}
		}
		if topic.HN > 0 {
			result = result.Add(Ballot{N: topic.HN, X: topic.B.X})
		}
	}
	switch topic := e.T.(type) {
	case *NomPrepTopic:
		f(&topic.PrepTopic)

	case *PrepTopic:
		f(topic)

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
	f := func(topic *PrepTopic) (bool, int, int) {
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
	}
	switch topic := e.T.(type) {
	case *NomPrepTopic:
		return f(&topic.PrepTopic)

	case *PrepTopic:
		return f(topic)

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

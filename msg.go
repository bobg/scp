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
	T *Topic      // The payload: a *NomTopic, a *NomPrepTopic, *PrepTopic, *CommitTopic, or *ExtTopic.
}

var msgCounter int32

// NewMsg produces a new message.
func NewMsg(v NodeID, i SlotID, q []NodeIDSet, t *Topic) *Msg {
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

	switch {
	case e.T.NomTopic != nil:
		err := nom(e.T.NomTopic)
		if err != nil {
			return err
		}

	case e.T.NomPrepTopic != nil:
		err := nom(&e.T.NomPrepTopic.NomTopic)
		if err != nil {
			return err
		}
		return prep(&e.T.NomPrepTopic.PrepTopic)

	case e.T.PrepTopic != nil:
		return prep(e.T.PrepTopic)

	case e.T.CommitTopic != nil:
		if e.T.CommitTopic.CN > e.T.CommitTopic.HN {
			return errors.New("CN > HN (commit)")
		}
	}
	return nil
}

// Return the ballot counter (if any).
func (e *Msg) bN() int {
	switch {
	case e.T.NomPrepTopic != nil:
		return e.T.NomPrepTopic.B.N

	case e.T.PrepTopic != nil:
		return e.T.PrepTopic.B.N

	case e.T.CommitTopic != nil:
		return e.T.CommitTopic.B.N

	case e.T.ExtTopic != nil:
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
	switch {
	case e.T.NomTopic != nil:
		f(e.T.NomTopic)
	case e.T.NomPrepTopic != nil:
		f(&e.T.NomPrepTopic.NomTopic)
	}
	return result
}

// Returns the set of values that e accepts as nominated.
func (e *Msg) acceptsNominatedSet() ValueSet {
	switch {
	case e.T.NomTopic != nil:
		return e.T.NomTopic.Y

	case e.T.NomPrepTopic != nil:
		return e.T.NomPrepTopic.Y
	}
	return nil
}

// Returns the set of ballots for which e votes or accepts "prepared."
func (e *Msg) votesOrAcceptsPreparedSet() BallotSet {
	result := e.acceptsPreparedSet()
	f := func(topic *PrepTopic) {
		result = result.Add(topic.B)
	}

	switch {
	case e.T.NomPrepTopic != nil:
		f(&e.T.NomPrepTopic.PrepTopic)

	case e.T.PrepTopic != nil:
		f(e.T.PrepTopic)

	case e.T.CommitTopic != nil:
		result = result.Add(Ballot{N: math.MaxInt32, X: e.T.CommitTopic.B.X})
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
	switch {
	case e.T.NomPrepTopic != nil:
		f(&e.T.NomPrepTopic.PrepTopic)

	case e.T.PrepTopic != nil:
		f(e.T.PrepTopic)

	case e.T.CommitTopic != nil:
		result = result.Add(Ballot{N: e.T.CommitTopic.PN, X: e.T.CommitTopic.B.X})
		result = result.Add(Ballot{N: e.T.CommitTopic.HN, X: e.T.CommitTopic.B.X})

	case e.T.ExtTopic != nil:
		result = result.Add(Ballot{N: math.MaxInt32, X: e.T.ExtTopic.C.X})
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
	switch {
	case e.T.NomPrepTopic != nil:
		return f(&e.T.NomPrepTopic.PrepTopic)

	case e.T.PrepTopic != nil:
		return f(e.T.PrepTopic)

	case e.T.CommitTopic != nil:
		if !ValueEqual(e.T.CommitTopic.B.X, v) {
			return false, 0, 0
		}
		if e.T.CommitTopic.CN > max {
			return false, 0, 0
		}
		if e.T.CommitTopic.CN > min {
			min = e.T.CommitTopic.CN
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
	switch {
	case e.T.CommitTopic != nil:
		if !ValueEqual(e.T.CommitTopic.B.X, v) {
			return false, 0, 0
		}
		if e.T.CommitTopic.CN > max {
			return false, 0, 0
		}
		if e.T.CommitTopic.HN < min {
			return false, 0, 0
		}
		if e.T.CommitTopic.CN > min {
			min = e.T.CommitTopic.CN
		}
		if e.T.CommitTopic.HN < max {
			max = e.T.CommitTopic.HN
		}
		return true, min, max

	case e.T.ExtTopic != nil:
		if !ValueEqual(e.T.ExtTopic.C.X, v) {
			return false, 0, 0
		}
		if e.T.ExtTopic.C.N > max {
			return false, 0, 0
		}
		if e.T.ExtTopic.C.N > min {
			min = e.T.ExtTopic.C.N
		}
		return true, min, max
	}
	return false, 0, 0
}

// String produces a readable representation of a message.
func (e *Msg) String() string {
	return fmt.Sprintf("(C=%d V=%s I=%d: %s)", e.C, e.V, e.I, e.T)
}

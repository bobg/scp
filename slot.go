package scp

import (
	"bytes"
	"math"
	"reflect"
	"time"
)

// SlotID is the type of a slot ID.
type SlotID int

// Slot maintains the state of a node's slot while it is undergoing
// nomination and balloting.
type Slot struct {
	ID   SlotID
	V    *Node
	Ph   Phase           // PhNom -> PhPrep -> PhCommit -> PhExt
	M    map[NodeID]*Msg // latest message from each peer
	sent Topic           // latest message sent

	T time.Time // time at which this slot was created (for computing the nomination round)
	X ValueSet  // votes for nominate(val)
	Y ValueSet  // votes for accept(nominate(val))
	Z ValueSet  // confirmed nominated values

	maxPriPeers NodeIDSet // set of peers that have ever had max priority
	nextRound   int       // 1+(latest round at which maxPriPeers was updated)

	B     Ballot
	P, PP Ballot // two highest "accepted prepared" ballots with differing values
	C, H  Ballot // lowest and highest confirmed-prepared or accepted-commit ballots (depending on phase)

	Upd *time.Timer // timer for invoking a deferred update
}

// Phase is the type of a slot's phase.
type Phase int

const (
	PhNom Phase = iota
	PhPrep
	PhCommit
	PhExt
)

func newSlot(id SlotID, n *Node) *Slot {
	return &Slot{
		ID: id,
		V:  n,
		T:  time.Now(),
		M:  make(map[NodeID]*Msg),
	}
}

var (
	// NomRoundInterval determines the duration of a nomination "round."
	// Round N lasts for a duration of (2+N)*NomRoundInterval.  A node's
	// neighbor set changes from one round to the next, as do the
	// priorities of the peers in that set.
	NomRoundInterval = time.Second

	// DeferredUpdateInterval determines the delay between arming a
	// deferred-update timer and firing it. The delay is
	// (1+N)*DeferredUpdateInterval, where N is the value of the slot's
	// ballot counter (B.N).
	DeferredUpdateInterval = time.Second
)

// This embodies most of the nomination and balloting protocols. It
// processes an incoming protocol message and returns an outbound
// protocol message in response, or nil if the incoming message is
// ignored.
// TODO: prune quorum searches when receiving a "confirm" vote. ("Once
// "v" enters the confirmed state, it may issue a _confirm_ "a"
// message to help other nodes confirm "a" more efficiently by pruning
// their quorum search at "v".")
func (s *Slot) handle(msg *Msg) (resp *Msg, err error) {
	if s.V.ID == msg.V && s.Ph != PhNom {
		// A node doesn't message itself except during nomination.
		return nil, nil
	}

	err = msg.valid()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil && resp != nil {
			if reflect.DeepEqual(resp.T, s.sent) {
				resp = nil
			} else {
				s.sent = resp.T
				s.Logf("%s -> %s", msg, resp)
			}
		}
	}()

	if have, ok := s.M[msg.V]; ok && !have.T.Less(msg.T) {
		// We already have a message from this sender that's the same or
		// newer; use that instead.
		msg = have
	} else {
		s.M[msg.V] = msg
	}

	if s.Ph == PhNom {
		err = s.doNomPhase(msg)
		if err != nil {
			return nil, err
		}
	}

	if s.Ph == PhPrep {
		err = s.doPrepPhase(msg)
		if err != nil {
			return nil, err
		}
	}
	if s.Ph == PhCommit {
		s.doCommitPhase()
	}

	return s.Msg(), nil
}

func (s *Slot) doNomPhase(msg *Msg) error {
	ok, err := s.maxPrioritySender(msg.V)
	if err != nil {
		return err
	}

	if ok {
		// "Echo" nominated values by adding them to s.X.
		switch topic := msg.T.(type) {
		case *NomTopic:
			s.X = s.X.Union(topic.X)
			s.X = s.X.Union(topic.Y)
		case *PrepTopic:
			s.X = s.X.Add(topic.B.X)
			if !topic.P.IsZero() {
				s.X = s.X.Add(topic.P.X)
			}
			if !topic.PP.IsZero() {
				s.X = s.X.Add(topic.PP.X)
			}
		case *CommitTopic:
			s.X = s.X.Add(topic.B.X)
		case *ExtTopic:
			s.X = s.X.Add(topic.C.X)
		}
	}

	// Promote accepted-nominated values from X to Y, and
	// confirmed-nominated values from Y to Z.
	s.updateYZ()

	if len(s.Z) > 0 {
		s.Ph = PhPrep
		s.B.N = 1
		s.setBX()
	}

	return nil
}

func (s *Slot) doPrepPhase(msg *Msg) error {
	if topic, ok := msg.T.(*NomTopic); ok {
		if s.H.N == 0 {
			// Can still update s.Z and s.B.X
			ok, err := s.maxPrioritySender(msg.V)
			if err != nil {
				return err
			}
			if ok {
				s.X = s.X.Union(topic.X)
				s.X = s.X.Union(topic.Y)
				s.updateYZ()
				s.B.X = s.Z.Combine(s.ID) // xxx does this require changing s.B.N?
			}
		}
		return nil
	}

	s.updateP()

	// Update s.H, the highest confirmed-prepared ballot.
	var cpIn, cpOut BallotSet
	if !s.P.IsZero() {
		cpIn = cpIn.Add(s.P)
		if !s.PP.IsZero() {
			cpIn = cpIn.Add(s.PP)
		}
	}
	nodeIDs := s.findQuorum(&ballotSetPred{
		ballots:      cpIn,
		finalBallots: &cpOut,
		testfn: func(msg *Msg, ballots BallotSet) BallotSet {
			return msg.acceptsPreparedSet()
		},
	})
	if len(nodeIDs) > 0 {
		h := cpOut[len(cpOut)-1]
		if s.H.Less(h) {
			s.H = h
		}
	}

	s.updateB()

	// Update s.C.
	if !s.C.IsZero() {
		if (s.C.Less(s.P) && !ValueEqual(s.P.X, s.C.X)) || (s.C.Less(s.PP) && !ValueEqual(s.PP.X, s.C.X)) {
			s.C = ZeroBallot
		}
	}
	if s.C.IsZero() && !s.H.IsZero() && !s.P.Aborts(s.H) && !s.PP.Aborts(s.H) {
		s.C = s.B
	}

	// The PREPARE phase ends at a node when the statement "commit
	// b" reaches the accept state in federated voting for some
	// ballot "b".
	if s.updateAcceptsCommitBounds() {
		// Accept commit(<n, s.B.X>).
		s.Ph = PhCommit
	}

	return nil
}

func (s *Slot) doCommitPhase() {
	s.updateP()
	s.updateAcceptsCommitBounds()
	s.updateB()

	// As soon as a node confirms "commit b" for any ballot "b", it
	// moves to the EXTERNALIZE stage.
	var cn, hn int
	nodeIDs := s.findQuorum(&minMaxPred{
		min:      s.C.N,
		max:      s.H.N,
		finalMin: &cn,
		finalMax: &hn,
		testfn: func(msg *Msg, min, max int) (bool, int, int) {
			return msg.acceptsCommit(s.B.X, min, max)
		},
	})
	if len(nodeIDs) > 0 {
		s.Ph = PhExt // \o/
		s.C.N = cn
		s.H.N = hn
		s.cancelUpd()
	}
}

func (s *Slot) updateAcceptsCommitBounds() bool {
	var cn, hn int
	nodeIDs := s.accept(func(isQuorum bool) predicate {
		return &minMaxPred{
			min:      1,
			max:      math.MaxInt32,
			finalMin: &cn,
			finalMax: &hn,
			testfn: func(msg *Msg, min, max int) (bool, int, int) {
				rangeFn := msg.acceptsCommit
				if isQuorum {
					rangeFn = msg.votesOrAcceptsCommit
				}
				return rangeFn(s.B.X, min, max)
			},
		}
	})
	if len(nodeIDs) > 0 {
		s.C.N = cn
		s.C.X = s.B.X
		s.H.N = hn
		s.H.X = s.B.X
		return true
	}
	return false
}

func (s *Slot) Msg() *Msg {
	msg := NewMsg(s.V.ID, s.ID, s.V.Q, nil)
	switch s.Ph {
	case PhNom:
		if len(s.X) == 0 && len(s.Y) == 0 {
			return nil
		}
		msg.T = &NomTopic{
			X: s.X,
			Y: s.Y,
		}

	case PhPrep:
		msg.T = &PrepTopic{
			B:  s.B,
			P:  s.P,
			PP: s.PP,
			HN: s.H.N,
			CN: s.C.N,
		}

	case PhCommit:
		msg.T = &CommitTopic{
			B:  s.B,
			PN: s.P.N,
			HN: s.H.N,
			CN: s.C.N,
		}

	case PhExt:
		msg.T = &ExtTopic{
			C:  s.C,
			HN: s.H.N,
		}
	}
	return msg
}

func (s *Slot) deferredUpdate() {
	if s.Upd == nil {
		return
	}

	s.Upd = nil
	s.B.N++
	s.setBX()

	s.Logf("deferred update, B is now %s", s.B)

	s.V.send <- s.Msg()
}

func (s *Slot) cancelUpd() {
	if s.Upd == nil {
		return
	}
	if !s.Upd.Stop() {
		// To prevent a timer created with NewTimer from firing after a
		// call to Stop, check the return value and drain the
		// channel. https://golang.org/pkg/time/#Timer.Stop
		<-s.Upd.C
	}
	s.Upd = nil
}

func (s *Slot) updateB() {
	// Update s.B.
	if s.B.Less(s.H) {
		// raise B to the highest confirmed-prepared ballot
		s.B = s.H
		s.cancelUpd()
		return
	}
	// When a node sees sees messages from a quorum to which it
	// belongs such that each message's "ballot.counter" is
	// greater than or equal to the local "ballot.counter", the
	// node arms a timer for its local "ballot.counter + 1"
	// seconds.
	if s.Upd == nil { // don't bother if a timer's already armed
		nodeIDs := s.findQuorum(fpred(func(msg *Msg) bool {
			return msg.bN() >= s.B.N
		}))
		if len(nodeIDs) > 0 {
			s.Upd = time.AfterFunc(time.Duration((1+s.B.N)*int(DeferredUpdateInterval)), s.deferredUpdate)
		}
	}

	// If nodes forming a blocking threshold all have
	// "ballot.counter" values greater than the local
	// "ballot.counter", then the local node immediately increases
	// "ballot.counter" to the lowest value such that this is no
	// longer the case.  (When doing so, it also disables any
	// pending timers associated with the old "counter".)
	//
	// TODO: new language says "if appropriate according to the rules
	// above arms a new timer"
	//
	// TODO: this code uses the minimum value from the blocking set
	// found, but should instead use the max-min from all possible
	// blocking sets (i.e., after this, there must be no blocking set
	// where all ballot counters are higher).
	//
	// TODO: the "to avoid exhausting ballot.counter" logic.
	var doSetBX bool
	for {
		nodeIDs := s.findBlockingSet(fpred(func(msg *Msg) bool {
			return msg.bN() > s.B.N
		}))
		if len(nodeIDs) == 0 {
			break
		}
		doSetBX = true
		s.cancelUpd()
		for i, nodeID := range nodeIDs {
			msg := s.M[nodeID]
			bn := msg.bN()
			if i == 0 || bn < s.B.N {
				s.B.N = bn
			}
		}
	}
	if doSetBX {
		s.setBX()
	}
}

func (s *Slot) setBX() {
	if s.Ph >= PhCommit {
		return
	}
	if !s.H.IsZero() {
		s.B.X = s.H.X
	} else {
		s.B.X = s.Z.Combine(s.ID)
	}
}

// Round tells the current (time-based) nomination round.
//
// Nomination round N lasts for a duration of
// (2+N)*NomRoundInterval. Also, the first round is round 1. Via the
// quadratic formula this tells us that after an elapsed time of T,
// it's round 1 + ((sqrt(8T+25)-5) / 2)
func (s *Slot) Round() int {
	return round(time.Since(s.T))
}

func round(d time.Duration) int {
	elapsed := float64(d) / float64(NomRoundInterval)
	r := math.Sqrt(8.0*elapsed + 25.0)
	return 1 + int((r-5.0)/2.0)
}

// Tells whether the given peer has or had the maximum priority in the
// current or any earlier nomination round.
func (s *Slot) maxPrioritySender(nodeID NodeID) (bool, error) {
	round := s.Round()
	for r := s.nextRound; r <= round; r++ {
		neighbors, err := s.V.Neighbors(s.ID, r)
		if err != nil {
			return false, err
		}
		var (
			maxPriority [32]byte
			sender      NodeID
		)
		for _, neighbor := range neighbors {
			priority, err := s.V.Priority(s.ID, r, neighbor)
			if err != nil {
				return false, err
			}
			if bytes.Compare(priority[:], maxPriority[:]) > 0 {
				maxPriority = priority
				sender = neighbor
			}
		}
		s.maxPriPeers = s.maxPriPeers.Add(sender)
	}
	s.nextRound = round + 1
	return s.maxPriPeers.Contains(nodeID), nil
}

func (s *Slot) updateYZ() {
	// Look for values to promote from s.X to s.Y.
	var promote ValueSet

	nodeIDs := s.accept(func(isQuorum bool) predicate {
		return &valueSetPred{
			vals:      s.X,
			finalVals: &promote,
			testfn: func(msg *Msg, vals ValueSet) ValueSet {
				setFn := msg.acceptsNominatedSet
				if isQuorum {
					setFn = msg.votesOrAcceptsNominatedSet
				}
				return vals.Intersection(setFn())
			},
		}
	})
	if len(nodeIDs) > 0 {
		s.Y = s.Y.Union(promote)
	}
	s.X = s.X.Minus(s.Y)

	// Look for values in s.Y to confirm, moving slot to the PREPARE
	// phase.
	promote = nil
	nodeIDs = s.findQuorum(&valueSetPred{
		vals:      s.Y,
		finalVals: &promote,
		testfn: func(msg *Msg, vals ValueSet) ValueSet {
			return vals.Intersection(msg.acceptsNominatedSet())
		},
	})
	if len(nodeIDs) > 0 {
		s.Z = s.Z.Union(promote)
	}
}

// Update s.P and s.PP, the two highest accepted-prepared ballots.
// TODO: this gives the highest accepted-prepared ballots in the
// blocking set or, if there isn't one, in the first quorum
// found. There might be higher accepted-prepared ballots in other
// quorums.
func (s *Slot) updateP() {
	s.P = ZeroBallot
	s.PP = ZeroBallot

	var apIn, apOut BallotSet
	peers := s.V.Peers()
	for _, peerID := range peers {
		if msg, ok := s.M[peerID]; ok {
			apIn = apIn.Union(msg.votesOrAcceptsPreparedSet())
		}
	}
	nodeIDs := s.accept(func(isQuorum bool) predicate {
		return &ballotSetPred{
			ballots:      apIn,
			finalBallots: &apOut,
			testfn: func(msg *Msg, ballots BallotSet) BallotSet {
				setFn := msg.acceptsPreparedSet
				if isQuorum {
					setFn = msg.votesOrAcceptsPreparedSet
				}
				return ballots.Intersection(setFn())
			},
		}
	})
	if len(nodeIDs) > 0 {
		s.P = apOut[len(apOut)-1]
		if s.Ph == PhPrep {
			for i := len(apOut) - 2; i >= 0; i-- {
				ap := apOut[i]
				if ap.N < s.P.N && !ValueEqual(ap.X, s.P.X) {
					s.PP = ap
					break
				}
			}
		}
	}
}

func (s *Slot) Logf(f string, a ...interface{}) {
	f = "slot %d: " + f
	a = append([]interface{}{s.ID}, a...)
	s.V.Logf(f, a...)
}

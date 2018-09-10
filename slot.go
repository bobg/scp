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
	Ph   Phase           // PhNom -> PhNomPrep -> PhPrep -> PhCommit -> PhExt
	M    map[NodeID]*Msg // latest message from each peer
	sent Topic           // latest message sent

	T time.Time // time at which this slot was created (for computing the nomination round)
	X ValueSet  // votes for nominate(val)
	Y ValueSet  // votes for accept(nominate(val))
	Z ValueSet  // confirmed nominated values

	maxPriPeers    NodeIDSet // set of peers that have ever had max priority
	lastRound      int       // latest round at which maxPriPeers was updated
	nextRoundTimer *time.Timer

	B     Ballot
	P, PP Ballot // two highest "accepted prepared" ballots with differing values
	C, H  Ballot // lowest and highest confirmed-prepared or accepted-commit ballots (depending on phase)

	Upd *time.Timer // timer for invoking a deferred update
}

// Phase is the type of a slot's phase.
type Phase int

const (
	PhNom Phase = iota
	PhNomPrep
	PhPrep
	PhCommit
	PhExt
)

func newSlot(id SlotID, n *Node) (*Slot, error) {
	s := &Slot{
		ID: id,
		V:  n,
		Ph: PhNom,
		T:  time.Now(),
		M:  make(map[NodeID]*Msg),
	}
	peerID, err := s.findMaxPriPeer(1)
	if err != nil {
		return nil, err
	}
	s.maxPriPeers = s.maxPriPeers.Add(peerID)
	s.lastRound = 1
	s.scheduleRound()
	return s, nil
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
	if s.V.ID == msg.V && !s.isNomPhase() {
		// A node doesn't message itself except during nomination.
		return nil, nil
	}

	err = msg.valid()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			if resp != nil {
				if reflect.DeepEqual(resp.T, s.sent) {
					resp = nil
				} else {
					s.sent = resp.T
				}
			}
			if resp != nil {
				s.Logf("%s -> %v", msg, resp)
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

	if s.isNomPhase() {
		s.doNomPhase(msg)
	}

	if s.isPrepPhase() {
		s.doPrepPhase()
	}

	if s.Ph == PhCommit {
		s.doCommitPhase()
	}

	return s.Msg(), nil
}

func (s *Slot) isNomPhase() bool {
	return s.Ph == PhNom || s.Ph == PhNomPrep
}

func (s *Slot) isPrepPhase() bool {
	return s.Ph == PhNomPrep || s.Ph == PhPrep
}

func (s *Slot) doNomPhase(msg *Msg) {
	if len(s.Z) == 0 && s.maxPrioritySender(msg.V) {
		// "Echo" nominated values by adding them to s.X.
		f := func(topic *NomTopic) {
			s.X = s.X.Union(topic.X)
			s.X = s.X.Union(topic.Y)
		}
		switch topic := msg.T.(type) {
		case *NomTopic:
			f(topic)
		case *NomPrepTopic:
			f(&topic.NomTopic)
		}
	}

	// Promote accepted-nominated values from X to Y, and
	// confirmed-nominated values from Y to Z.
	s.updateYZ()

	if s.Ph == PhNom {
		if len(s.Z) > 0 {
			// Some value is confirmed nominated, start PREPARE phase.
			s.Ph = PhNomPrep
			s.B.N = 1
			s.setBX()
		} else {
			s.updateP()
			if !s.P.IsZero() {
				s.Ph = PhNomPrep
				s.B.N = 1
				s.setBX()
			}
		}
	}
}

func (s *Slot) doPrepPhase() {
	s.updateP() // xxx may be redundant with the call in doNomPhase

	// Update s.H, the highest confirmed-prepared ballot.
	s.H = ZeroBallot
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
			return ballots.Intersection(msg.acceptsPreparedSet())
		},
	})
	if len(nodeIDs) > 0 {
		h := cpOut[len(cpOut)-1]
		if ValueEqual(s.B.X, h.X) {
			s.H = h
		}
		if s.Ph == PhNomPrep {
			// Some ballot is confirmed prepared, exit NOMINATE phase.
			s.Ph = PhPrep
			s.cancelRounds()
		}
	}

	s.updateB()

	// Update s.C.
	if !s.C.IsZero() {
		if s.H.N == 0 || (s.C.Less(s.P) && !ValueEqual(s.P.X, s.C.X)) || (s.C.Less(s.PP) && !ValueEqual(s.PP.X, s.C.X)) {
			s.C = ZeroBallot
		}
	}
	if s.C.IsZero() && s.H.N > 0 && s.H.N == s.B.N {
		s.C = s.B
	}

	// The PREPARE phase ends at a node when the statement "commit
	// b" reaches the accept state in federated voting for some
	// ballot "b".
	if s.updateAcceptsCommitBounds() {
		// Accept commit(<n, s.B.X>).
		s.Ph = PhCommit
	}
}

func (s *Slot) doCommitPhase() {
	s.cancelRounds()
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

func (s *Slot) cancelRounds() {
	if s.nextRoundTimer == nil {
		return
	}
	stopTimer(s.nextRoundTimer)
	s.nextRoundTimer = nil
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

	case PhNomPrep:
		msg.T = &NomPrepTopic{
			NomTopic: NomTopic{
				X: s.X,
				Y: s.Y,
			},
			PrepTopic: PrepTopic{
				B:  s.B,
				P:  s.P,
				PP: s.PP,
				HN: s.H.N,
				CN: s.C.N,
			},
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

// "When a node sees sees messages from a quorum to which it belongs
// such that each message's "ballot.counter" is greater than or equal
// to the local "ballot.counter", the node arms a timer for its local
// "ballot.counter + 1" seconds."
func (s *Slot) maybeScheduleUpd() {
	if s.Upd != nil {
		// Don't bother if a timer's already armed.
		return
	}
	nodeIDs := s.findQuorum(fpred(func(msg *Msg) bool {
		return msg.bN() >= s.B.N
	}))
	if len(nodeIDs) == 0 {
		return
	}
	s.Upd = time.AfterFunc(time.Duration((1+s.B.N)*int(DeferredUpdateInterval)), func() {
		s.V.deferredUpdate(s)
	})
}

func (s *Slot) deferredUpdate() {
	if s.Upd == nil {
		return
	}

	s.Upd = nil
	s.B.N++
	s.setBX()

	if s.isPrepPhase() {
		s.doPrepPhase()
	}
	if s.Ph == PhCommit {
		s.doCommitPhase()
	}

	msg := s.Msg()

	s.Logf("deferred update: %s", msg)

	s.V.send <- msg
}

func (s *Slot) cancelUpd() {
	if s.Upd == nil {
		return
	}
	stopTimer(s.Upd)
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

	s.maybeScheduleUpd()

	// If nodes forming a blocking threshold all have
	// "ballot.counter" values greater than the local
	// "ballot.counter", then the local node immediately increases
	// "ballot.counter" to the lowest value such that this is no
	// longer the case.  (When doing so, it also disables any
	// pending timers associated with the old "counter".)
	var (
		doSetBX bool
		setBN   = s.B.N
	)
	for { // loop until no such blocking set is found
		nodeIDs := s.findBlockingSet(fpred(func(msg *Msg) bool {
			return msg.bN() > setBN
		}))
		if len(nodeIDs) == 0 {
			break
		}

		doSetBX = true
		s.cancelUpd()
		var innerSetBN int
		for i, nodeID := range nodeIDs {
			msg := s.M[nodeID]
			bn := msg.bN()
			if i == 0 || bn < innerSetBN {
				innerSetBN = bn
			}
		}
		if innerSetBN > setBN {
			setBN = innerSetBN
		}
	}

	if setBN == s.B.N {
		return
	}

	// To avoid exhausting `ballot.counter`, its value must always be
	// less then 1,000 plus the number of seconds a node has been
	// running SCP on the current slot.  Should any of the above rules
	// require increasing the counter beyond this value, a node either
	// increases `ballot.counter` to the maximum permissible value,
	// or, if it is already at this maximum, waits up to one second
	// before increasing the value.
	maxBN := 1000 + int(time.Since(s.T)/time.Second)
	if setBN <= maxBN {
		s.B.N = setBN
	} else if s.B.N < maxBN {
		s.Logf("limiting B.N to %d (from %d)", maxBN, setBN)
		s.B.N = maxBN
	} else {
		setBN = maxBN + 1

		// The time when it's ok to set s.B.N to setBN (i.e., after it's been running for setBN-1000 seconds)
		oktime := s.T.Add(time.Duration(setBN-1000) * time.Second)
		until := time.Until(oktime)

		s.Logf("limiting B.N to %d after a %s sleep", setBN, until)
		time.Sleep(until)
		s.B.N = setBN
	}
	if doSetBX {
		s.setBX()
		s.maybeScheduleUpd()
	}
}

func (s *Slot) setBX() {
	if s.Ph >= PhCommit {
		return
	}
	switch {
	case !s.H.IsZero():
		s.B.X = s.H.X

	case len(s.Z) > 0:
		s.B.X = s.Z.Combine(s.ID)

	case !s.P.IsZero():
		s.B.X = s.P.X
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

func (s *Slot) roundTime(r int) time.Time {
	r--
	intervals := r * (r + 5) / 2
	return s.T.Add(time.Duration(intervals * int(NomRoundInterval)))
}

func (s *Slot) newRound() error {
	if s.nextRoundTimer == nil {
		return nil
	}

	curRound := s.Round()

	for r := s.lastRound + 1; r <= curRound; r++ {
		peerID, err := s.findMaxPriPeer(r)
		if err != nil {
			return err
		}
		s.maxPriPeers = s.maxPriPeers.Add(peerID)
	}
	// s.Logf("round %d, peers %v", curRound, s.maxPriPeers)
	s.lastRound = curRound
	s.V.rehandle(s)
	s.scheduleRound()
	return nil
}

func (s *Slot) scheduleRound() {
	dur := time.Until(s.roundTime(s.lastRound + 1))
	// s.Logf("scheduling round %d for %s from now", s.lastRound+1, dur)
	s.nextRoundTimer = time.AfterFunc(dur, func() {
		s.V.newRound(s)
	})
}

func (s *Slot) findMaxPriPeer(r int) (NodeID, error) {
	neighbors, err := s.V.Neighbors(s.ID, r)
	if err != nil {
		return "", err
	}
	var (
		maxPriority [32]byte
		result      NodeID
	)
	for _, neighbor := range neighbors {
		priority, err := s.V.Priority(s.ID, r, neighbor)
		if err != nil {
			return "", err
		}
		if bytes.Compare(priority[:], maxPriority[:]) > 0 {
			maxPriority = priority
			result = neighbor
		}
	}
	return result, nil
}

// Tells whether the given peer has or had the maximum priority in the
// current or any earlier nomination round.
func (s *Slot) maxPrioritySender(nodeID NodeID) bool {
	return s.maxPriPeers.Contains(nodeID)
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
		if !s.B.IsZero() {
			// Exclude ballots with N > B.N, if s.B is set.
			// If it's not set, we're still in NOMINATE phase and can set
			// s.P to anything.
			for len(apOut) > 0 && apOut[len(apOut)-1].N > s.B.N {
				apOut = apOut[:len(apOut)-1]
			}
		}
		if len(apOut) > 0 {
			s.P = apOut[len(apOut)-1]
			if !s.B.IsZero() && s.P.N == s.B.N && s.B.X.Less(s.P.X) {
				s.P.N--
			}
			if s.Ph == PhPrep {
				s.PP = ZeroBallot
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
}

func (s *Slot) Logf(f string, a ...interface{}) {
	f = "slot %d: " + f
	a = append([]interface{}{s.ID}, a...)
	s.V.Logf(f, a...)
}

// To prevent a timer created with NewTimer from firing after a call
// to Stop, check the return value and drain the
// channel. https://golang.org/pkg/time/#Timer.Stop
//
// HOWEVER, it looks like a straight read of the timer's channel can
// sometimes block even when Stop returns false. This works around
// that by making the drain be non-blocking.
func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

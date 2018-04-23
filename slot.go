package scp

import (
	"bytes"
	"math"
	"time"
)

// SlotID is the type of a slot ID.
type SlotID int

// Slot maintains the state of a node's slot while it is undergoing
// nomination and balloting.
type Slot struct {
	ID SlotID
	V  *Node
	Ph Phase           // PhNom -> PhPrep -> PhCommit -> PhExt
	M  map[NodeID]*Msg // latest message from each peer

	T time.Time // time at which this slot was created (for computing the nomination round)
	X ValueSet  // votes for nominate(val)
	Y ValueSet  // votes for accept(nominate(val))
	Z ValueSet  // confirmed nominated values

	B      Ballot
	P, PP  Ballot    // two highest "prepared" ballots with differing values
	C, H   Ballot    // lowest and highest confirmed-prepared or accepted-commit ballots (depending on phase)
	AP, CP BallotSet // accepted-prepared, confirmed-prepared

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

// Handle embodies most of the nomination and balloting protocols. It
// processes an incoming protocol message and returns an outbound
// protocol message in response, or nil if the incoming message is
// ignored.
func (s *Slot) Handle(msg *Msg) (resp *Msg, err error) {
	if have, ok := s.M[msg.V]; ok && !have.T.Less(msg.T) && s.Ph != PhNom {
		// We already have a message from this sender that's the same or
		// newer.
		return nil, nil
	}

	s.M[msg.V] = msg

	switch s.Ph { // note, s.Ph == PhExt should never be true
	case PhNom:
		if ok, err := s.maxPrioritySender(msg.V); err != nil || !ok {
			return nil, err
		}

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

		// Promote accepted-nominated values from X to Y, and
		// confirmed-nominated values from Y to Z.
		s.updateYZ()

		if len(s.Z) > 0 {
			s.Ph = PhPrep
			s.B.N = 1
			s.setBX()
		}

	case PhPrep:
		if topic, ok := msg.T.(*NomTopic); ok {
			if s.H.N == 0 {
				// Can still update s.Z and s.B.X
				ok, err := s.maxPrioritySender(msg.V)
				if err != nil {
					return nil, err
				}
				if ok {
					s.X = s.X.Union(topic.X)
					s.X = s.X.Union(topic.Y)
					s.updateYZ()
					s.B.X = s.Z.Combine()
				}
			}
		} else {
			s.updateAP()

			// Update s.P and s.PP, the two highest accepted-prepared
			// ballots with unequal values.
			if len(s.AP) > 0 {
				s.P = s.AP[len(s.AP)-1]
				s.PP = ZeroBallot
				for i := len(s.AP) - 2; i >= 0; i-- {
					ap := s.AP[i]
					if ap.N < s.P.N && !VEqual(ap.X, s.P.X) {
						s.PP = ap
						break
					}
				}
			}

			// Update s.CP, the set of confirmed-prepared ballots.
			var cps []Ballot
			for _, ap := range s.AP {
				if s.CP.Contains(ap) {
					continue
				}
				nodeIDs := s.findQuorum(fpred(func(msg *Msg) bool {
					return msg.acceptsPrepared(ap)
				}))
				if len(nodeIDs) > 0 {
					cps = append(cps, ap)
				}
			}
			for _, cp := range cps {
				s.AP = s.AP.Remove(cp)
				s.CP = s.CP.Add(cp)
			}

			// Update s.H, the highest confirmed-prepared ballot.
			if len(s.CP) > 0 && s.H.Less(s.CP[len(s.CP)-1]) {
				s.H = s.CP[len(s.CP)-1]
			}

			// Update s.B.
			if s.B.Less(s.H) {
				// raise B to the highest confirmed-prepared ballot
				s.B = s.H
				s.cancelUpd()
			} else {
				s.updateB()
			}

			// Update s.C.
			if !s.C.IsZero() {
				if (s.C.Less(s.P) && !VEqual(s.P.X, s.C.X)) || (s.C.Less(s.PP) && !VEqual(s.PP.X, s.C.X)) {
					s.C = ZeroBallot
				}
			}
			if s.C.IsZero() && s.H.N > 0 && !s.P.Aborts(s.H) && !s.PP.Aborts(s.H) {
				s.C = s.B
			}

			// The PREPARE phase ends at a node when the statement "commit
			// b" reaches the accept state in federated voting for some
			// ballot "b".
			if !s.C.IsZero() && !s.H.IsZero() {
				var cn, hn int
				pred := &minMaxPred{
					min:      s.C.N,
					max:      s.H.N,
					finalMin: &cn,
					finalMax: &hn,
					testfn: func(msg *Msg, min, max int) (bool, int, int) {
						return msg.votesOrAcceptsCommit(s.B.X, min, max)
					},
				}
				nodeIDs := s.findBlockingSetOrQuorum(pred)
				if len(nodeIDs) > 0 {
					// There is a blocking set or quorum that votes-or-accepts
					// commit(<n, s.B.X>) for various ranges of n that have a
					// non-empty overlap, so we can accept commit(<n, s.B.X>).
					s.Ph = PhCommit
					s.C.N = cn
					s.H.N = hn
				}
			}
		}

	case PhCommit:
		s.updateAP()
		s.P = s.AP[len(s.AP)-1]

		// Update the accepted-commit bounds.
		var acmin, acmax int
		acpred := &minMaxPred{
			min:      s.C.N,
			max:      math.MaxInt32,
			finalMin: &acmin,
			finalMax: &acmax,
			testfn: func(msg *Msg, min, max int) (bool, int, int) {
				return msg.votesOrAcceptsCommit(s.B.X, min, max)
			},
		}
		nodeIDs := s.findBlockingSetOrQuorum(acpred)
		if len(nodeIDs) > 0 {
			s.C.N = acmin
			s.H.N = acmax
		}

		// As soon as a node confirms "commit b" for any ballot "b", it
		// moves to the EXTERNALIZE stage.
		var cn, hn int
		ccpred := &minMaxPred{
			min:      s.C.N,
			max:      s.H.N,
			finalMin: &cn,
			finalMax: &hn,
			testfn: func(msg *Msg, min, max int) (bool, int, int) {
				return msg.acceptsCommit(s.B.X, min, max)
			},
		}
		nodeIDs = s.findQuorum(ccpred)
		if len(nodeIDs) > 0 {
			s.Ph = PhExt // \o/
			s.C.N = cn
			s.H.N = hn
		}
	}

	// Compute a response message.
	msg = NewMsg(s.V.ID, s.ID, s.V.Q, nil)
	switch s.Ph {
	case PhNom:
		if len(s.X) == 0 && len(s.Y) == 0 {
			return nil, nil
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

	return msg, nil
}

func (s *Slot) deferredUpdate() {
	s.V.mu.Lock() // xxx maybe this needs to be a Node method
	defer s.V.mu.Unlock()

	if s.Upd == nil {
		return
	}

	s.Upd = nil
	s.B.N++
	s.setBX()

	s.Logf("deferred update, B is now %s", s.B)
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

func (s *Slot) setBX() {
	if s.Ph >= PhCommit {
		return
	}
	if len(s.CP) > 0 {
		s.B.X = s.CP[len(s.CP)-1].X
	} else {
		s.B.X = s.Z.Combine()
	}
}

// Round tells the current (time-based) nomination round.
//
// Nomination round N lasts for a duration of
// (2+N)*NomRoundInterval. Via the quadratic formula this tells us
// that after an elapsed time of T, it's round sqrt(1+T)-1.
func (s *Slot) Round() int {
	elapsed := float64(time.Since(s.T)) / float64(NomRoundInterval)
	return int(math.Sqrt(1.0+elapsed) - 1.0)
}

// Tells whether the given peer has or had the maximum priority in the
// current or any earlier nomination round.
func (s *Slot) maxPrioritySender(nodeID NodeID) (bool, error) {
	for round := s.Round(); round >= 0; round-- {
		neighbors, err := s.V.Neighbors(s.ID, round)
		if err != nil {
			return false, err
		}
		var (
			maxPriority [32]byte
			sender      NodeID
		)
		for _, neighbor := range neighbors {
			priority, err := s.V.Priority(s.ID, round, neighbor)
			if err != nil {
				return false, err
			}
			if bytes.Compare(priority[:], maxPriority[:]) > 0 {
				maxPriority = priority
				sender = neighbor
			}
		}
		if sender == nodeID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Slot) updateYZ() {
	// Look for values to promote from s.X to s.Y.
	// xxx there is surely a better way to do this
	var promote ValueSet
	for _, val := range s.X {
		nodeIDs := s.findBlockingSetOrQuorum(fpred(func(msg *Msg) bool {
			return msg.votesOrAcceptsNominated(val)
		}))
		if len(nodeIDs) > 0 {
			promote = promote.Add(val)
		}
	}
	for _, val := range promote {
		s.X = s.X.Remove(val)
		s.Y = s.Y.Add(val)
	}

	// Look for values in s.Y to confirm, moving slot to the PREPARE
	// phase.
	for _, val := range s.Y {
		nodeIDs := s.findQuorum(fpred(func(msg *Msg) bool {
			return msg.acceptsNominated(val)
		}))
		if len(nodeIDs) > 0 {
			s.Z = s.Z.Add(val)
		}
	}
}

// Update s.AP - the set of accepted-prepared ballots.
func (s *Slot) updateAP() {
	if !s.AP.Contains(s.B) {
		nodeIDs := s.findBlockingSetOrQuorum(fpred(func(msg *Msg) bool {
			return msg.votesOrAcceptsPrepared(s.B)
		}))
		if len(nodeIDs) > 0 {
			s.AP = s.AP.Add(s.B)
		}
	}
}

func (s *Slot) updateB() {
	// When a node sees sees messages from a quorum to which it
	// belongs such that each message's "ballot.counter" is
	// greater than or equal to the local "ballot.counter", the
	// node arms a timer for its local "ballot.counter + 1"
	// seconds.
	if s.Upd == nil { // don't bother if a timer's already armed
		nodeIDs := s.findQuorum(fpred(func(msg *Msg) bool {
			return msg.T.BN() > s.B.N
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
	nodeIDs := s.findBlockingSet(fpred(func(msg *Msg) bool {
		return msg.T.BN() > s.B.N
	}))
	if len(nodeIDs) > 0 {
		s.cancelUpd()
		for i, nodeID := range nodeIDs {
			msg := s.M[nodeID]
			bn := msg.T.BN()
			if i == 0 || bn < s.B.N {
				s.B.N = bn
			}
		}
		s.setBX()
	}
}

func (s *Slot) Logf(f string, a ...interface{}) {
	f = "slot %d: " + f
	a = append([]interface{}{s.ID}, a...)
	s.V.Logf(f, a...)
}

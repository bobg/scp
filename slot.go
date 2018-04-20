package scp

import (
	"bytes"
	"time"
)

type SlotID int

type Slot struct {
	ID SlotID
	Ph Phase
	V  *Node
	T  time.Time       // time at which this slot was created
	M  map[NodeID]*Env // latest message from each peer

	X ValueSet // votes for nominate(val)
	Y ValueSet // votes for accept(nominate(val))
	Z ValueSet // confirmed nominated values

	B      Ballot
	C      Ballot
	AP, CP []Ballot // accepted-prepared, confirmed-prepared; kept sorted
	HN     int

	Upd *time.Timer
}

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
		M:  make(map[NodeID]*Env),
	}
}

const (
	roundDuration          = 100 * time.Millisecond
	deferredUpdateInterval = 250 * time.Millisecond
)

func (s *Slot) Handle(env *Env) (*Env, error) {
	if have, ok := s.M[env.V]; ok && !have.Less(env) {
		// We already have a message from this sender that's the same or
		// newer.
		return nil, nil
	}

	s.M[env.V] = env

	switch s.Ph { // note, s.Ph == PhExt should never be true
	case PhNom:
		switch msg := env.M.(type) {
		case *NomMsg:
			// nom nom
			err := s.handleNomMsg(msg)
			if err != nil {
				return nil, err
			}

		case *PrepMsg:
			// Prep msg in nom phase
			// B.X, P.X, and PPrime.X are all accepted-nominated by env.V
			s.X.Add(msg.B.X)
			if !msg.P.IsZero() {
				s.X.Add(msg.P.X)
			}
			if !msg.PPrime.IsZero() {
				s.X.Add(msg.PPrime.X)
			}

		case *CommitMsg:
			// Commit msg in nom phase
			// B.X is accepted-nominated by env.V
			s.X.Add(msg.B.X)

		case *ExtMsg:
			// Ext msg in nom phase
			// C.X is accepted-nominated by env.V
			s.X.Add(msg.C.X)
		}

		s.updateXYZ()

		if len(s.Z) > 0 {
			s.Ph = PhPrep
			s.B.N = 1
			s.setBX()
		}

	case PhPrep:
		if msg, ok := env.M.(*NomMsg); ok && s.HN == 0 {
			// Can still update s.Z and s.B.X
			err := s.handleNomMsg(env, msg)
			if err != nil {
				return nil, err
			}
			s.updateXYZ()
			s.B.X = s.Z.Combine()
		} else {
			// xxx update s.AP, s.CP, s.C, s.HN
			if len(s.CP) > 0 && s.B.Less(s.CP[len(s.CP)-1]) {
				// raise B to the highest confirmed-prepared ballot
				s.B = s.CP[len(s.CP)-1]
				s.cancelUpd()
			} else {
				// When a node sees sees messages from a quorum to which it
				// belongs such that each message's "ballot.counter" is
				// greater than or equal to the local "ballot.counter", the
				// node arms a timer for its local "ballot.counter + 1"
				// seconds.
				if s.Upd == nil { // don't bother if a timer's already armed
					nodeIDs := s.findQuorum(func(env *Env) bool {
						return env.M.BN() > s.B.N
					})
					if len(nodeIDs) > 0 {
						s.Upd = time.AfterFunc((1+s.B.N)*deferredUpdateInterval, s.deferredUpdate)
					}
				}

				// If nodes forming a blocking threshold all have
				// "ballot.counter" values greater than the local
				// "ballot.counter", then the local node immediately increases
				// "ballot.counter" to the lowest value such that this is no
				// longer the case.  (When doing so, it also disables any
				// pending timers associated with the old "counter".)
				nodeIDs := s.findBlockingSet(func(env *Env) bool {
					return env.M.BN() > s.B.N
				})
				if len(nodeIDs) > 0 {
					s.cancelUpd()
					for i, nodeID := range nodeIDs {
						env := s.M[nodeID]
						bn := env.M.BN()
						if i == 0 || bn < s.B.N {
							s.B.N = bn
						}
					}
					s.setBX()
				}
			}

			// xxx set p and pp (p')

			if !s.C.IsZero() && ((!p.IsZero() && s.C.Less(p) && !VEqual(p.X, s.C.X)) || (!pp.IsZero() && s.C.Less(pp) && !VEqual(pp.X, s.C.X))) {
				s.C = ZeroBallot
			}
			if s.C.IsZero() && s.HN > 0 {
				// Check h is not aborted by p or pp.
				if (p.IsZero() || !p.Aborts(h)) && (pp.IsZero() || !pp.Aborts(h)) {
					s.C = s.B
				}
			}

			// The PREPARE phase ends at a node when the statement "commit
			// b" reaches the accept state in federated voting for some
			// ballot "b".
			if !s.C.IsZero() {
				pred := &acceptCommitPredicate{min: s.C.N, max: s.HN, val: s.B.X}
				nodeIDs := s.findBlockingSetOrQuorum(pred)
				if len(nodeIDs) > 0 {
					// There is a blocking set or quorum that votes-or-accepts
					// commit(<n, s.B.X>) for various ranges of n that have a
					// non-empty overlap, so we can accept commit(<n, s.B.X>).
					s.Ph = PhCommit
				}
			}
		}

	case PhCommit:
		// xxx update s.AP, s.CP, s.C, s.HN
		if s.HN > s.B.N {
			s.B.N = s.HN
		}
		// xxx update accepted-commit ballots

		// As soon as a node confirms "commit b" for any ballot "b", it
		// moves to the EXTERNALIZE stage.
		pred := &confirmCommitPredicate{min: s.C.N, max: s.HN, val: s.B.X}
		nodeIDs := s.findQuorum(pred)
		if len(nodeIDs) > 0 {
			s.Ph = PhExt // \o/
		}
	}

	// Compute a response message.
	env := &Env{
		V: s.V.ID,
		I: s.ID,
		Q: s.V.Q,
	}
	switch s.Ph {
	case PhNom:
		if len(s.X) == 0 && len(s.Y) == 0 {
			return nil, nil
		}
		env.M = &NomMsg{
			X: s.X,
			Y: s.Y,
		}

	case PhPrep:
		env.M = &PrepMsg{
			B:  s.B,
			P:  s.P,
			PP: s.PP,
			HN: s.H.N,
			CN: s.C.N,
		}

	case PhCommit:
		env.M = &CommitMsg{
			B:  s.B,
			PN: s.P.N,
			HN: s.H.N,
			CN: s.C.N,
		}

	case PhExt:
		env.M = &ExtMsg{
			C:  s.C,
			HN: s.H.N,
		}
	}

	return env, nil
}

func (s *Slot) deferredUpdate() {
	s.V.mu.Lock()
	defer s.V.mu.Unlock()

	s.Upd = nil
	s.B.N++
	s.setBX()
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
	if len(s.CP) > 0 {
		s.B.X = s.CP[len(s.CP)-1].X
	} else {
		s.B.X = s.Z.Combine()
	}
}

func (s *Slot) handleNomMsg(env *Env, msg *NomMsg) error {
	round := int(time.Since(s.T) / roundDuration)
	neighbors, err := s.V.Neighbors(s.ID, round)
	if err != nil {
		return err
	}

	var (
		maxPriority          [32]byte
		senderHasMaxPriority bool
	)
	for _, neighbor := range neighbors {
		priority, err := s.V.Priority(s.ID, round, neighbor)
		if err != nil {
			return err
		}
		if bytes.Compare(priority[:], maxPriority[:]) > 0 {
			maxPriority = priority
			senderHasMaxPriority = (neighbor == env.V)
		}
	}
	if senderHasMaxPriority {
		s.X.AddSet(msg.X)
		s.X.AddSet(msg.Y)
	}
	return nil
}

func (s *Slot) updateXYZ() {
	// Look for values to promote from s.X to s.Y.
	// xxx there is surely a better way to do this
	var promote ValueSet
	for _, val := range s.X {
		nodeIDs := s.findBlockingSetOrQuorum(func(env *Env) bool {
			switch msg := env.M.(type) {
			case *NomMsg:
				return msg.X.Contains(val) || msg.Y.Contains(val)

			case *PrepMsg:
				return VEqual(msg.B.X, val) || VEqual(msg.P.X, val) || VEqual(msg.PPrime.X, val)

			case *CommitMsg:
				return VEqual(msg.B.X, val)

			case *ExtMsg:
				return VEqual(msg.C.X, val)
			}
			return false // not reached
		})
		if len(nodeIDs) > 0 {
			promote.Add(val)
		}
	}
	for _, val := range promote {
		s.X.Remove(val)
		s.Y.Add(val)
	}

	// Look for values in s.Y to confirm, moving slot to the PREPARE
	// phase.
	for _, val := range s.Y {
		nodeIDs := s.findQuorum(func(env *Env) bool {
			switch msg := env.M.(type) {
			case *NomMsg:
				return s.Y.Contains(val)

			case *PrepMsg:
				return VEqual(msg.B.X, val) || VEqual(msg.P.X, val) || VEqual(msg.PPrime.X, val)

			case *CommitMsg:
				return VEqual(msg.B.X, val)

			case *ExtMsg:
				return VEqual(msg.C.X, val)
			}
			return false // not reached
		})
		if len(nodeIDs) > 0 {
			s.Z.Add(val)
		}
	}
}

type acceptCommitPredicate struct {
	min, max         int
	val              Value
	nextMin, nextMax int
}

func (p *acceptCommitPredicate) test(env *Env) bool {
	p.nextMin, p.nextMax = p.min, p.max
	if p.min > p.max {
		return false
	}
	switch msg := env.M.(type) {
	case *PrepMsg:
		if msg.CN == 0 || !VEqual(msg.B.X, p.val) {
			return false
		}
		if msg.CN > p.max || msg.HN < p.min {
			return false
		}
		if msg.CN > p.min {
			p.nextMin = msg.CN
		}
		if msg.HN < msg.max {
			p.nextMax = msg.HN
		}
		return true

	case *CommitMsg:
		if !VEqual(msg.B.X, p.val) {
			return false
		}
		if msg.CN > p.max {
			return false
		}
		if msg.CN > p.min {
			p.nextMin = msg.CN
		}
		return true

	case *ExtMsg:
		if !VEqual(msg.C.X, p.val) {
			return false
		}
		if msg.C.N > p.max {
			return false
		}
		if msg.C.N > p.min {
			p.nextMin = msg.C.N
		}
		return true
	}

	return false
}

func (p *acceptCommitPredicate) next() predicate {
	return &acceptCommitPredicate{
		min: p.nextMin,
		max: p.nextMax,
		val: p.val,
	}
}

type confirmCommitPredicate struct {
	min, max         int
	val              Value
	nextMin, nextMax int
}

func (p *confirmCommitPredicate) test(env *Env) bool {
	p.nextMin, p.nextMax = p.min, p.max
	if p.min > p.max {
		return false
	}
	switch msg := env.M.(type) {
	case *CommitMsg:
		if !VEqual(msg.B.X, p.val) {
			return false
		}
		if msg.CN > p.max || msg.HN < p.min {
			return false
		}
		if msg.CN > p.min {
			p.nextMin = msg.CN
		}
		if msg.HN < p.max {
			p.nextMax = msg.HN
		}
		return true

	case *ExtMsg:
		if !VEqual(msg.C.X, p.val) {
			return false
		}
		if msg.C.N > p.max {
			return false
		}
		if msg.C.N > p.min {
			p.nextMin = msg.C.N
		}
		return true
	}
	return false
}

func (p *confirmCommitPredicate) next() predicate {
	return &confirmCommitPredicate{
		min: p.nextMin,
		max: p.nextMax,
		val: p.val,
	}
}

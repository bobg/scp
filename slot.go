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

	B      Ballot
	AP, CP []Ballot // accepted-prepared, confirmed-prepared; kept sorted
	HN, CN int
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

const roundDuration = 100 * time.Millisecond

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
			round := int(time.Since(s.T) / roundDuration)
			neighbors, err := s.V.Neighbors(s.ID, round)
			if err != nil {
				return nil, err
			}

			var (
				maxPriority          [32]byte
				senderHasMaxPriority bool
			)
			for _, neighbor := range neighbors {
				priority, err := s.V.Priority(s.ID, round, neighbor)
				if err != nil {
					return nil, err
				}
				if bytes.Compare(priority[:], maxPriority[:]) > 0 {
					maxPriority = priority
					senderHasMaxPriority = (neighbor == env.V)
				}
			}
			if !senderHasMaxPriority {
				return nil, nil
			}
			s.X.AddSet(msg.X)
			s.X.AddSet(msg.Y)

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

		// Look for values to promote from s.X to s.Y.
		// xxx there is surely a better way to do this
		var promote ValueSet
		for _, val := range s.X {
			if s.blockingSetOrQuorumExists(func(nodeID NodeID) bool {
				env, ok := s.M[nodeID]
				if !ok {
					return false
				}
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
			}) {
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
			nodeIDs := s.findQuorum(func(nodeID NodeID) bool {
				env, ok := s.M[nodeID]
				if !ok {
					return false
				}
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
				s.Ph = PhPrep
				s.B.N = 1
				s.B.X = val
				break
			}
		}

	case PhPrep:
		switch msg := env.M.(type) {
		case *NomMsg:
		case *PrepMsg:
		case *CommitMsg:
		case *ExtMsg:
		}

	case PhCommit:
		switch msg := env.M.(type) {
		case *NomMsg:
		case *PrepMsg:
		case *CommitMsg:
		case *ExtMsg:
		}
	}
}

func (s *Slot) blockingSetOrQuorumExists(pred func(NodeID) bool) bool {
	nodeIDs := s.findBlockingSet(pred)
	if len(nodeIDs) > 0 {
		return true
	}
	nodeIDs = s.findQuorum(pred)
	return len(nodeIDs) > 0
}

func (s *Slot) findBlockingSet(pred func(NodeID) bool) []NodeID {
	var result []NodeID
	for _, slice := range s.V.Q {
		var found bool
		for _, nodeID := range slice {
			if pred(nodeID) {
				result = append(result, nodeID)
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return result
}

// findQuorum finds a quorum containing the slot's node in which every
// node satisfies the given predicate.
func (s *Slot) findQuorum(pred func(NodeID) bool) []NodeID {
	m := make(map[NodeID]struct{})
	m = s.findNodeQuorum(s.V.ID, s.V.Q, pred, m)
	if len(m) == 0 {
		return nil
	}
	result := make([]NodeID, 0, len(m))
	for n := range m {
		result = append(result, n)
	}
	return result
}

// findNodeQuorum is a helper function for findQuorum. It checks that
// the node has at least one slice whose members (and the transitive
// closure over them) all satisfy the given predicate.
func (s *Slot) findNodeQuorum(nodeID NodeID, q QSet, pred func(NodeID) bool, m map[NodeID]struct{}) map[NodeID]struct{} {
	for _, slice := range q {
		m2 := s.findSliceQuorum(slice, pred, m)
		if len(m2) > 0 {
			return m2
		}
	}
	return nil
}

// findSliceQuorum is a helper function for findNodeQuorum. It checks
// whether every node in a given quorum slice (and the transitive
// closure over them) satisfies the given predicate.
func (s *Slot) findSliceQuorum(slice []NodeID, pred func(NodeID) bool, m map[NodeID]struct{}) map[NodeID]struct{} {
	var newNodeIDs []NodeID
	for _, nodeID := range slice {
		if _, ok := m[nodeID]; !ok {
			newNodeIDs = append(newNodeIDs, nodeID)
		}
	}
	if len(newNodeIDs) == 0 {
		return m
	}
	for _, nodeID := range newNodeIDs {
		if !pred(nodeID) {
			return nil
		}
	}
	m2 := make(map[NodeID]struct{})
	for nodeID := range m {
		m2[nodeID] = struct{}{}
	}
	for _, nodeID := range newNodeIDs {
		m2[nodeID] = struct{}{}
	}
	for _, nodeID := range newNodeIDs {
		env, ok := s.M[nodeID]
		if !ok {
			return nil
		}
		m2 = s.findNodeQuorum(nodeID, env.Q, pred, m2)
		if len(m2) == 0 {
			return nil
		}
	}
	return m2
}

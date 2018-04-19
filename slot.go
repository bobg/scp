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
			if senderHasMaxPriority {
				s.X.AddSet(msg.X)
				s.X.AddSet(msg.Y)

				// xxx look for values to promote from s.X to s.Y
				// xxx look for values in s.Y to confirm, moving slot to the PREPARE phase
			}

		case *PrepMsg:
			// xxx prep msg in nom phase - B.X, P.X, and PPrime.X are all
			// accepted-nominated by env.V

		case *CommitMsg:
			// xxx commit msg in nom phase - B.X is accepted-nominated by
			// env.V

		case *ExtMsg:
			// xxx ext msg in nom phase - C.X is accepted-nominated by env.V
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

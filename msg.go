package scp

type Msg interface {
	BN() int // returns ballot.counter in PREPARE and COMMIT messages
	Less(Msg) bool
}

// NomMsg is the payload of a nomination protocol message.
type NomMsg struct {
	X, Y ValueSet
}

func (nm *NomMsg) BN() int { return 0 }

func (nm *NomMsg) Less(other Msg) bool {
	if other, ok := other.(*NomMsg); ok {
		// xxx
	}
	return true
}

type PrepMsg struct {
	B, P, PPrime Ballot
	HN, CN       int
}

func (pm *PrepMsg) BN() int { return pm.B.N }

func (pm *PrepMsg) Less(other Msg) bool {
	switch other := other.(type) {
	case *NomMsg:
		return false
	case *PrepMsg:
		// xxx
	}
	return true
}

type CommitMsg struct {
	B          Ballot
	PN, HN, CN int
}

func (cm *CommitMsg) BN() int { return cm.B.N }

func (cm *CommitMsg) Less(other Msg) bool {
	switch other := other.(type) {
	case *NomMsg:
		return false
	case *PrepMsg:
		return false
	case *CommitMsg:
		// xxx
	}
	return true
}

type ExtMsg struct {
	C  Ballot
	HN int
}

func (em *ExtMsg) BN() int { return 0 }

func (em *ExtMsg) Less(other Msg) bool {
	if other, ok := other.(*ExtMsg); ok {
		// xxx
	}
	return false
}

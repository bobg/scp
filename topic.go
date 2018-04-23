package scp

import "fmt"

// Topic is the abstract type of the payload of an SCP message
// (conveyed in an envelope, see type Msg).
type Topic interface {
	BN() int // returns ballot.counter in PREPARE and COMMIT messages
	Less(Topic) bool
	String() string
}

// NomTopic is the payload of a nomination protocol message.
type NomTopic struct {
	X, Y ValueSet
}

func (nt *NomTopic) BN() int { return 0 }

func (nt *NomTopic) Less(other Topic) bool {
	o, ok := other.(*NomTopic)
	if !ok {
		return true // NOMINATE messages are less than all other messages
	}
	return len(nt.X)+len(nt.Y) < len(o.X)+len(o.Y)
}

func (nt *NomTopic) String() string {
	return fmt.Sprintf("NOM X=%s, Y=%s", nt.X, nt.Y)
}

// PrepTopic is the payload of a PREPARE message in the ballot protocol.
type PrepTopic struct {
	B, P, PP Ballot
	HN, CN   int
}

func (pt *PrepTopic) BN() int { return pt.B.N }

func (pt *PrepTopic) Less(other Topic) bool {
	switch other := other.(type) {
	case *NomTopic:
		return false
	case *PrepTopic:
		if pt.B.Less(other.B) {
			return true
		}
		if other.B.Less(pt.B) {
			return false
		}
		if pt.P.Less(other.P) {
			return true
		}
		if other.P.Less(pt.P) {
			return false
		}
		if pt.PP.Less(other.PP) {
			return true
		}
		if other.PP.Less(pt.PP) {
			return false
		}
		return pt.HN < other.HN
	}
	return true
}

func (pt *PrepTopic) String() string {
	return fmt.Sprintf("PREP B=%s P=%s PP=%s CN=%d HN=%d", pt.B, pt.P, pt.PP, pt.CN, pt.HN)
}

// CommitTopic is the payload of a COMMIT message in the ballot
// protocol.
type CommitTopic struct {
	B          Ballot
	PN, HN, CN int
}

func (ct *CommitTopic) BN() int { return ct.B.N }

func (ct *CommitTopic) Less(other Topic) bool {
	switch other := other.(type) {
	case *NomTopic:
		return false
	case *PrepTopic:
		return false
	case *CommitTopic:
		if ct.B.Less(other.B) {
			return true
		}
		if other.B.Less(ct.B) {
			return false
		}
		if ct.PN < other.PN {
			return true
		}
		if other.PN < ct.PN {
			return false
		}
		return ct.HN < other.HN
	}
	return true
}

func (ct *CommitTopic) String() string {
	return fmt.Sprintf("COMMIT B=%s PN=%d CN=%d HN=%d", ct.B, ct.PN, ct.CN, ct.HN)
}

// ExtTopic is the payload of an EXTERNALIZE message in the ballot
// protocol.
type ExtTopic struct {
	C  Ballot
	HN int
}

func (et *ExtTopic) BN() int { return 0 }

func (et *ExtTopic) Less(other Topic) bool {
	if other, ok := other.(*ExtTopic); ok {
		return et.HN < other.HN
	}
	return false
}

func (et *ExtTopic) String() string {
	return fmt.Sprintf("EXT C=%s HN=%d", et.C, et.HN)
}

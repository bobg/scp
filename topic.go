package scp

import "fmt"

// Topic is the type of the payload of an SCP message
// (conveyed in an envelope, see type Msg).
// Exactly one of its fields is non-nil.
type Topic struct {
	NomTopic     *NomTopic     `json:"nom_topic,omitempty"`
	NomPrepTopic *NomPrepTopic `json:"nom_prep_topic,omitempty"`
	PrepTopic    *PrepTopic    `json:"prep_topic,omitempty"`
	CommitTopic  *CommitTopic  `json:"commit_topic,omitempty"`
	ExtTopic     *ExtTopic     `json:"ext_topic,omitempty"`
}

func (t *Topic) Less(other *Topic) bool {
	switch {
	case t.NomTopic != nil:
		return t.NomTopic.Less(other)
	case t.NomPrepTopic != nil:
		return t.NomPrepTopic.Less(other)
	case t.PrepTopic != nil:
		return t.PrepTopic.Less(other)
	case t.CommitTopic != nil:
		return t.CommitTopic.Less(other)
	case t.ExtTopic != nil:
		return t.ExtTopic.Less(other)
	}
	return false
}

func (t *Topic) String() string {
	switch {
	case t.NomTopic != nil:
		return t.NomTopic.String()
	case t.NomPrepTopic != nil:
		return t.NomPrepTopic.String()
	case t.PrepTopic != nil:
		return t.PrepTopic.String()
	case t.CommitTopic != nil:
		return t.CommitTopic.String()
	case t.ExtTopic != nil:
		return t.ExtTopic.String()
	}
	return ""
}

// NomTopic is the payload of a nomination protocol message.
type NomTopic struct {
	X, Y ValueSet
}

func (nt *NomTopic) Less(other *Topic) bool {
	o := other.NomTopic
	if o == nil {
		return true // NOMINATE messages are less than all other messages
	}
	if len(nt.Y) < len(o.Y) {
		return true
	}
	if len(nt.Y) > len(o.Y) {
		return false
	}
	return len(nt.X) < len(o.X)
}

func (nt *NomTopic) String() string {
	return fmt.Sprintf("NOM X=%s, Y=%s", nt.X, nt.Y)
}

// NomPrepTopic is the combined payload of a NOMINATE and a PREPARE
// message.
type NomPrepTopic struct {
	NomTopic
	PrepTopic
}

func (npt *NomPrepTopic) Less(other *Topic) bool {
	switch {
	case other.NomTopic != nil:
		return false

	case other.NomPrepTopic != nil:
		if npt.NomTopic.Less(&Topic{NomTopic: &other.NomPrepTopic.NomTopic}) {
			return true
		}
		if other.NomPrepTopic.NomTopic.Less(&Topic{NomTopic: &npt.NomTopic}) {
			return false
		}
		return npt.PrepTopic.Less(&Topic{PrepTopic: &other.NomPrepTopic.PrepTopic})

	default:
		return true
	}
}

func (npt *NomPrepTopic) String() string {
	return fmt.Sprintf("NOM/PREP X=%s, Y=%s B=%s P=%s PP=%s CN=%d HN=%d", npt.X, npt.Y, npt.B, npt.P, npt.PP, npt.CN, npt.HN)
}

// PrepTopic is the payload of a PREPARE message in the ballot protocol.
type PrepTopic struct {
	B, P, PP Ballot
	HN, CN   int
}

func (pt *PrepTopic) Less(other *Topic) bool {
	switch {
	case other.NomTopic != nil:
		return false
	case other.NomPrepTopic != nil:
		return false
	case other.PrepTopic != nil:
		if pt.B.Less(other.PrepTopic.B) {
			return true
		}
		if other.PrepTopic.B.Less(pt.B) {
			return false
		}
		if pt.P.Less(other.PrepTopic.P) {
			return true
		}
		if other.PrepTopic.P.Less(pt.P) {
			return false
		}
		if pt.PP.Less(other.PrepTopic.PP) {
			return true
		}
		if other.PrepTopic.PP.Less(pt.PP) {
			return false
		}
		return pt.HN < other.PrepTopic.HN
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

func (ct *CommitTopic) Less(other *Topic) bool {
	switch {
	case other.NomTopic != nil:
		return false
	case other.NomPrepTopic != nil:
		return false
	case other.PrepTopic != nil:
		return false
	case other.CommitTopic != nil:
		if ct.B.Less(other.CommitTopic.B) {
			return true
		}
		if other.CommitTopic.B.Less(ct.B) {
			return false
		}
		if ct.PN < other.CommitTopic.PN {
			return true
		}
		if other.CommitTopic.PN < ct.PN {
			return false
		}
		return ct.HN < other.CommitTopic.HN
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

func (et *ExtTopic) Less(other *Topic) bool {
	if other := other.ExtTopic; other != nil {
		return et.HN < other.HN
	}
	return false
}

func (et *ExtTopic) String() string {
	return fmt.Sprintf("EXT C=%s HN=%d", et.C, et.HN)
}

package scp

import (
	"fmt"
)

// Ballot is an SCP ballot.
type Ballot struct {
	N int
	X Value
}

// ZeroBallot is the zero ballot.
var ZeroBallot Ballot

// IsZero tells whether b is the zero ballot.
func (b Ballot) IsZero() bool {
	return b.N == 0 && isNilVal(b.X)
}

// Less tells whether a ballot is less than another.
func (b Ballot) Less(other Ballot) bool {
	if b.N < other.N {
		return true
	}
	if b.N > other.N {
		return false
	}
	if isNilVal(b.X) {
		return !isNilVal(other.X)
	}
	if isNilVal(other.X) {
		return false
	}
	return b.X.Less(other.X)
}

// Equal tells whether a ballot is equal to another.
func (b Ballot) Equal(other Ballot) bool {
	return b.N == other.N && ValueEqual(b.X, other.X)
}

// String produces a readable representation of a ballot.
func (b Ballot) String() string {
	if b.IsZero() {
		return "<>"
	}
	return fmt.Sprintf("<%d,%s>", b.N, VString(b.X))
}

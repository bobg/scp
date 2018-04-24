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
	return b.N == 0 && b.X == nil
}

// Less tells whether a ballot is less than another.
func (b Ballot) Less(other Ballot) bool {
	if b.N < other.N {
		return true
	}
	if b.N > other.N {
		return false
	}
	if b.X == nil {
		return other.X != nil
	}
	if other.X == nil {
		return false
	}
	return b.X.Less(other.X)
}

// Equal tells whether a ballot is equal to another.
func (b Ballot) Equal(other Ballot) bool {
	return b.N == other.N && ValueEqual(b.X, other.X)
}

// Aborts tells whether a vote to prepare one ballot aborts another.
func (b Ballot) Aborts(other Ballot) bool {
	return other.N < b.N && !ValueEqual(other.X, b.X)
}

// String produces a readable representation of a ballot.
func (b Ballot) String() string {
	if b.IsZero() {
		return "<>"
	}
	return fmt.Sprintf("<%d,%s>", b.N, VString(b.X))
}

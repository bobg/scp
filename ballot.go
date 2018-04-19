package scp

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

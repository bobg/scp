package scp

import (
	"fmt"
	"sort"
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
	return b.N == other.N && VEqual(b.X, other.X)
}

// Aborts tells whether a vote to prepare one ballot aborts another.
func (b Ballot) Aborts(other Ballot) bool {
	return other.N < b.N && !VEqual(other.X, b.X)
}

// String produces a readable representation of a ballot.
func (b Ballot) String() string {
	if b.IsZero() {
		return "<>"
	}
	return fmt.Sprintf("<%d,%s>", b.N, VString(b.X))
}

// BallotSet is a set of ballots, implemented as a sorted slice.
type BallotSet []Ballot

func (bs BallotSet) find(b Ballot) int {
	return sort.Search(len(bs), func(n int) bool {
		return !bs[n].Less(b)
	})
}

func (bs BallotSet) Add(b Ballot) BallotSet {
	index := bs.find(b)
	if index < len(bs) && b.Equal(bs[index]) {
		return bs
	}
	var result BallotSet
	result = append(result, bs[:index]...)
	result = append(result, b)
	result = append(result, bs[index:]...)
	return result
}

func (bs BallotSet) Remove(b Ballot) BallotSet {
	index := bs.find(b)
	if index >= len(bs) || !b.Equal(bs[index]) {
		return bs
	}
	var result BallotSet
	result = append(result, bs[:index]...)
	result = append(result, bs[index+1:]...)
	return result
}

func (bs BallotSet) Contains(b Ballot) bool {
	index := bs.find(b)
	return index < len(bs) && b.Equal(bs[index])
}

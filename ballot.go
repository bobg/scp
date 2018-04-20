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

func (b Ballot) Equal(other Ballot) bool {
	return b.N == other.N && VEqual(b.X, other.X)
}

// Aborts tells whether a vote to prepare b aborts other.
func (b Ballot) Aborts(other Ballot) bool {
	return other.N < b.N && !VEqual(other.X, b.X)
}

func (b Ballot) String() string {
	if b.IsZero() {
		return "<nil>"
	}
	return fmt.Sprintf("<%d,%s>", b.N, VString(b.X))
}

// BallotSet is a set of ballots, implemented as a sorted slice.
// TODO: merge with ValueSet
type BallotSet []Ballot

func (bs *BallotSet) Add(b Ballot) {
	if bs.Contains(b) {
		return
	}
	*bs = append(*bs, b)
	sort.Slice(*bs, func(i, j int) bool {
		return (*bs)[i].Less((*bs)[j])
	})
}

func (bs *BallotSet) Remove(b Ballot) {
	for i, elt := range *bs {
		if elt.Less(b) {
			continue
		}
		if b.Less(elt) {
			return
		}
		before := (*bs)[:i]
		after := (*bs)[i+1:]
		*bs = append([]Ballot{}, before...)
		*bs = append(*bs, after...)
		return
	}
}

func (bs BallotSet) Contains(b Ballot) bool {
	if len(bs) == 0 {
		return false
	}
	mid := len(bs) / 2
	if bs[mid].Less(b) {
		return bs[mid+1:].Contains(b)
	}
	if b.Less(bs[mid]) {
		if mid == 0 {
			return false
		}
		return bs[:mid-1].Contains(b)
	}
	return true
}

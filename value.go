package scp

import (
	"fmt"
	"sort"
	"strings"
)

// Value is the type of values being voted on by the network.
type Value interface {
	Less(Value) bool
	Combine(Value) Value
	Bytes() []byte
	String() string
}

func VEqual(a, b Value) bool {
	return !a.Less(b) && !b.Less(a)
}

func VString(v Value) string {
	if v == nil {
		return "<nil>"
	}
	return v.String()
}

// ValueSet is a set of values, implemented as a sorted slice.
type ValueSet []Value

// Add adds a Value to a ValueSet.
// TODO: this can be done in better than O(n log n).
func (vs *ValueSet) Add(v Value) {
	if vs.Contains(v) {
		return
	}
	*vs = append(*vs, v)
	sort.Slice(*vs, func(i, j int) bool {
		return (*vs)[i].Less((*vs)[j])
	})
}

// AddSet adds the members of one ValueSet to another.
// TODO: this can be done _much_ better.
func (vs *ValueSet) AddSet(other ValueSet) {
	for _, v := range other {
		vs.Add(v)
	}
}

// Remove removes a value from a set.
// TODO: it's almost like I don't have a CS degree or something.
func (vs *ValueSet) Remove(v Value) {
	for i, elt := range *vs {
		if elt.Less(v) {
			continue
		}
		if v.Less(elt) {
			return
		}
		before := (*vs)[:i]
		after := (*vs)[i+1:]
		*vs = append([]Value{}, before...)
		*vs = append(*vs, after...)
		return
	}
}

// Contains uses binary search to test whether vs contains v.
func (vs ValueSet) Contains(v Value) bool {
	if len(vs) == 0 {
		return false
	}
	mid := len(vs) / 2
	if vs[mid].Less(v) {
		return vs[mid+1:].Contains(v)
	}
	if v.Less(vs[mid]) {
		if mid == 0 {
			return false
		}
		return vs[:mid-1].Contains(v)
	}
	return true
}

// Combine reduces the members of vs to a single value using
// Value.Combine. The result is nil if vs is empty.
func (vs ValueSet) Combine() Value {
	if len(vs) == 0 {
		return nil
	}
	result := vs[0]
	for _, v := range vs[1:] {
		result = result.Combine(v)
	}
	return result
}

func (vs ValueSet) String() string {
	var strs []string
	for _, v := range vs {
		strs = append(strs, VString(v))
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, " "))
}

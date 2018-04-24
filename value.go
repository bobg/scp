package scp

import (
	"fmt"
	"sort"
	"strings"
)

// Value is the abstract type of values being voted on by the network.
type Value interface {
	// Less tells whether this value is less than another. Values must be totally ordered.
	Less(Value) bool

	// Combine combines this value with another to produce a third
	// (which may be the same as either of the inputs). The operation
	// should be deterministic and commutative.
	Combine(Value) Value

	// Bytes produces a byte-string representation of the value, not
	// meant for human consumption.
	Bytes() []byte

	// String produces a readable representation of the value.
	String() string
}

// VEqual tells whether two values are equal.
func VEqual(a, b Value) bool {
	if a == nil {
		return b == nil
	}
	if b == nil {
		return false
	}
	return !a.Less(b) && !b.Less(a)
}

// VString calls a Value's String method. If the value is nil, returns
// the string "<nil>".
func VString(v Value) string {
	if v == nil {
		return "<nil>"
	}
	return v.String()
}

// ValueSet is a set of values, implemented as a sorted slice.
type ValueSet []Value

func (vs ValueSet) find(v Value) int {
	return sort.Search(len(vs), func(n int) bool {
		return !vs[n].Less(v)
	})
}

// Add produces a ValueSet containing the members of vs plus the
// element v.
func (vs ValueSet) Add(v Value) ValueSet {
	index := vs.find(v)
	if index < len(vs) && VEqual(v, vs[index]) {
		return vs
	}
	var result ValueSet
	result = append(result, vs[:index]...)
	result = append(result, v)
	result = append(result, vs[index:]...)
	return result
}

// Union produces a ValueSet containing all the members of both sets.
func (vs ValueSet) Union(other ValueSet) ValueSet {
	if len(vs) == 0 {
		return other
	}
	if len(other) == 0 {
		return vs
	}
	var (
		i, j   int
		result ValueSet
	)
	for i < len(vs) && j < len(other) {
		switch {
		case vs[i].Less(other[j]):
			result = append(result, vs[i])
			i++
		case other[j].Less(vs[i]):
			result = append(result, other[j])
			j++
		default:
			result = append(result, vs[i])
			i++
			j++
		}
	}
	result = append(result, vs[i:]...)
	result = append(result, other[j:]...)
	return result
}

// Intersection produces a ValueSet with only the elements in both
// sets.
func (vs ValueSet) Intersection(other ValueSet) ValueSet {
	if len(vs) == 0 || len(other) == 0 {
		return nil
	}
	var result ValueSet
	for i, j := 0, 0; i < len(vs) && j < len(other); {
		switch {
		case vs[i].Less(other[j]):
			i++
		case other[j].Less(vs[i]):
			j++
		default:
			result = append(result, vs[i])
			i++
			j++
		}
	}
	return result
}

// Minus produces a ValueSet with only the members of vs that don't
// appear in other.
func (vs ValueSet) Minus(other ValueSet) ValueSet {
	if len(vs) == 0 || len(other) == 0 {
		return vs
	}
	var (
		result ValueSet
		i, j   int
	)
	for i < len(vs) && j < len(other) {
		switch {
		case vs[i].Less(other[j]):
			result = append(result, vs[i])
			i++
		case other[j].Less(vs[i]):
			j++
		default:
			i++
			j++
		}
	}
	result = append(result, vs[i:]...)
	return result
}

// Remove produces a ValueSet without the specified element.
func (vs ValueSet) Remove(v Value) ValueSet {
	index := vs.find(v)
	if index >= len(vs) || !VEqual(v, vs[index]) {
		return vs
	}
	var result ValueSet
	result = append(result, vs[:index]...)
	result = append(result, vs[index+1:]...)
	return result
}

// Contains tests whether vs contains v.
func (vs ValueSet) Contains(v Value) bool {
	index := vs.find(v)
	return index < len(vs) && VEqual(vs[index], v)
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

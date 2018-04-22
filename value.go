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
	if a == nil {
		return b == nil
	}
	if b == nil {
		return false
	}
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

func (vs ValueSet) find(v Value) int {
	return sort.Search(len(vs), func(n int) bool {
		return !vs[n].Less(v)
	})
}

// Add adds a Value to a ValueSet.
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

// AddSet adds the members of one ValueSet to another.
func (vs ValueSet) AddSet(other ValueSet) ValueSet {
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

// Remove removes a value from a set.
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

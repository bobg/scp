package scp

import (
	"fmt"
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

// VString calls a Value's String method. If the value is nil, returns
// the string "<nil>".
func VString(v Value) string {
	if v == nil {
		return "<nil>"
	}
	return v.String()
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

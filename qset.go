package scp

import (
	"bytes"
	"fmt"
	"math/big"
)

type (
	// QSet is a compact representation for a set of quorum slices.
	// A quorum slice is any T items from M,
	// where 0 < T <= len(M).
	// An item in M is either a node or a nested QSet.
	// If the latter,
	// any of the recursively defined subslices count as one "item" here.
	QSet struct {
		T int          `json:"threshold"`
		M []QSetMember `json:"members"`
	}

	// QSetMember is a member of a QSet.
	// It's either a node ID or a nested QSet.
	// Exactly one of its fields is non-nil.
	QSetMember struct {
		N *NodeID `json:"node_id,omitempty"`
		Q *QSet   `json:"qset,omitempty"`
	}
)

// Checks that at least one node in each quorum slice satisfies pred
// (excluding the slot's node).
//
// Works by finding len(q.M)-q.T+1 members for which pred is true
func (q QSet) findBlockingSet(msgs map[NodeID]*Msg, pred predicate) (NodeIDSet, predicate) {
	return findBlockingSetHelper(len(q.M)-q.T+1, q.M, msgs, pred, nil)
}

func findBlockingSetHelper(needed int, members []QSetMember, msgs map[NodeID]*Msg, pred predicate, sofar NodeIDSet) (NodeIDSet, predicate) {
	if needed == 0 {
		return sofar, pred
	}
	if needed > len(members) {
		return nil, pred
	}
	m0 := members[0]
	switch {
	case m0.N != nil:
		if msg, ok := msgs[*m0.N]; ok && pred.test(msg) {
			return findBlockingSetHelper(needed-1, members[1:], msgs, pred.next(), sofar.Add(*m0.N))
		}

	case m0.Q != nil:
		sofar2, pred2 := findBlockingSetHelper(len(m0.Q.M)-m0.Q.T+1, m0.Q.M, msgs, pred, sofar)
		if len(sofar2) > 0 {
			return findBlockingSetHelper(needed-1, members[1:], msgs, pred2, sofar2)
		}
	}
	return findBlockingSetHelper(needed, members[1:], msgs, pred, sofar)
}

// Finds a quorum in which every node satisfies the given
// predicate. The slot's node itself is presumed to satisfy the
// predicate.
func (q QSet) findQuorum(nodeID NodeID, m map[NodeID]*Msg, pred predicate) (NodeIDSet, predicate) {
	return findQuorumHelper(q.T, q.M, m, pred, NodeIDSet{nodeID})
}

func findQuorumHelper(threshold int, members []QSetMember, msgs map[NodeID]*Msg, pred predicate, sofar NodeIDSet) (NodeIDSet, predicate) {
	if threshold == 0 {
		return sofar, pred
	}
	if threshold > len(members) {
		return nil, pred
	}
	m0 := members[0]
	switch {
	case m0.N != nil:
		if sofar.Contains(*m0.N) {
			return findQuorumHelper(threshold-1, members[1:], msgs, pred, sofar)
		}
		if msg, ok := msgs[*m0.N]; ok && pred.test(msg) {
			return findQuorumHelper(threshold-1, members[1:], msgs, pred.next(), sofar.Add(*m0.N))
		}

	case m0.Q != nil:
		sofar2, pred2 := findQuorumHelper(m0.Q.T, m0.Q.M, msgs, pred, sofar)
		if len(sofar2) > 0 {
			return findQuorumHelper(threshold-1, members[1:], msgs, pred2, sofar2)
		}
	}
	return findQuorumHelper(threshold, members[1:], msgs, pred, sofar)
}

// Function weight returns the fraction of q's quorum slices in which id appears.
// Return value is the fraction and
// (as an optimization)
// a bool indicating whether it's exactly 1.
func (q QSet) weight(id NodeID) (float64, bool) {
	num, denom := q.NodeFrac(id)
	if num == denom {
		return 1.0, true
	}
	return float64(num) / float64(denom), false
}

// Slices calls f once for each slice represented by q.
// It continues until all slices have been generated or f returns false to terminate early.
func (q QSet) Slices(f func(NodeIDSet) bool) {
	slicesHelper(q.T, q.M, f, nil, 0)
}

func slicesHelper(t int, members []QSetMember, f func(NodeIDSet) bool, sofar NodeIDSet, depth int) (out bool) {
	if t == 0 {
		return f(sofar)
	}
	if t > len(members) {
		return true
	}

	m0 := members[0]
	switch {
	case m0.N != nil:
		if !slicesHelper(t-1, members[1:], f, append(sofar, *m0.N), depth+1) {
			return false
		}

	case m0.Q != nil:
		ok := slicesHelper(
			m0.Q.T,
			m0.Q.M,
			func(slice NodeIDSet) bool {
				return slicesHelper(t-1, members[1:], f, sofar.Union(slice), depth+1)
			},
			sofar,
			depth+1,
		)
		if !ok {
			return false
		}
	}
	return slicesHelper(t, members[1:], f, sofar, depth+1)
}

// Nodes returns a flattened set of all nodes contained in q and its nested QSets.
func (q QSet) Nodes() NodeIDSet {
	var result NodeIDSet

	for _, m := range q.M {
		switch {
		case m.N != nil:
			result = result.Add(*m.N)

		case m.Q != nil:
			result = result.Union(m.Q.Nodes())
		}
	}

	return result
}

func (q QSet) NumSlices() *big.Int {
	result := new(big.Int)
	result.Binomial(int64(len(q.M)), int64(q.T))
	for _, m := range q.M {
		if m.Q != nil {
			result.Mul(result, m.Q.NumSlices())
		}
	}
	return result
}

// NodeFrac gives the fraction of slices in q containing the given node.
// It assumes that id appears in at most one QSet
// (either the top level one or a single reachable nested one)
// and then only once in that QSet.
func (q QSet) NodeFrac(id NodeID) (num, denom int) {
	for _, m := range q.M {
		switch {
		case m.N != nil:
			if *m.N == id {
				return q.T, len(q.M)
			}

		case m.Q != nil:
			num2, denom2 := m.Q.NodeFrac(id)
			if num2 > 0 {
				return q.T * num2, len(q.M) * denom2
			}
		}
	}
	return 0, 1
}

func (m QSetMember) String() string {
	switch {
	case m.N != nil:
		return fmt.Sprintf("N:%s", *m.N)

	case m.Q != nil:
		b := new(bytes.Buffer)
		fmt.Fprintf(b, "Q:{T=%d [", m.Q.T)
		for i, mm := range m.Q.M {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(mm.String())
		}
		fmt.Fprint(b, "]}")
		return b.String()
	}
	return ""
}

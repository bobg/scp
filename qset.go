package scp

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
func (q QSet) findBlockingSet(msgs map[NodeID]*Msg, pred predicate) (NodeIDSet, predicate) {
	var result NodeIDSet

	memo := make(map[NodeID]bool)

	q.slices(func(slice NodeIDSet) bool {
		found := false
		for _, nodeID := range slice {
			if outcome, ok := memo[nodeID]; ok {
				if outcome {
					found = true
					break
				}
				continue
			}
			if msg, ok := msgs[nodeID]; ok {
				outcome := pred.test(msg)
				memo[nodeID] = outcome
				if outcome {
					pred = pred.next()
					found = true
					result = result.Add(nodeID)
					break
				}
			}
		}
		if !found {
			result = nil
		}
		return found
	})

	return result, pred
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
	if len(members) == 0 {
		return nil, pred
	}
	m0 := members[0]
	switch {
	case m0.N != nil:
		if sofar.Contains(*m0.N) {
			return findQuorumHelper(threshold-1, members[1:], msgs, pred, sofar)
		}
		if msg, ok := msgs[*m0.N]; ok && pred.test(msg) {
			sofar2, pred2 := findQuorumHelper(msg.Q.T, msg.Q.M, msgs, pred.next(), sofar.Add(*m0.N))
			if len(sofar2) > 0 {
				return findQuorumHelper(threshold-1, members[1:], msgs, pred2, sofar2)
			}
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
	// TODO: this implementation generates all the slices and tests each for the presence of id.
	// A smarter implementation would use math.
	// In a set of N nodes that contains id,
	// with threshold T,
	// weight is: (N-1 choose T-1) / (N choose T), which is T/N.
	// But I'm less sure about the math when nested QSets are involved.

	var num, denom int
	q.slices(func(n NodeIDSet) bool {
		denom++
		if n.Contains(id) {
			num++
		}
		return true
	})

	if num == denom {
		return 1.0, true
	}
	return float64(num) / float64(denom), false
}

func (q QSet) slices(f func(NodeIDSet) bool) {
	slicesHelper(q.T, q.M, f, nil)
}

// t > 0
func slicesHelper(t int, members []QSetMember, f func(NodeIDSet) bool, sofar NodeIDSet) {
	if t > len(members) {
		return
	}

	if t == 1 {
		for _, m := range members {
			switch {
			case m.N != nil:
				if !f(sofar.Add(*m.N)) {
					return
				}

			case m.Q != nil:
				slicesHelper(m.Q.T, m.Q.M, f, sofar)
			}
		}
		return
	}

	m0 := members[0]
	switch {
	case m0.N != nil:
		slicesHelper(t-1, members[1:], f, append(sofar, *m0.N))

	case m0.Q != nil:
		slicesHelper(
			m0.Q.T,
			m0.Q.M,
			func(n NodeIDSet) bool {
				slicesHelper(t-1, members[1:], f, sofar.Union(n))
				return true
			},
			sofar,
		)
	}
	slicesHelper(t, members[1:], f, sofar)
}

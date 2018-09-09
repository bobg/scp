package scp

type (
	QSet struct {
		Threshold int
		Members   []QSetMember
	}

	// QSetMember is a member of a QSet.
	// It's either a node ID or a nested QSet.
	// Exactly one of its fields is non-nil.
	QSetMember struct {
		NodeID *NodeID
		QSet   *QSet
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
	return findQuorumHelper(q.Threshold, q.Members, m, pred, NodeIDSet{nodeID})
}

func findQuorumHelper(threshold int, members []QSetMember, msgs map[NodeID]*Msg, pred predicate, sofar NodeIDSet) (NodeIDSet, predicate) {
	// xxx threshold == 0
	// xxx len(members) == 0
	m0 := members[0]
	switch {
	case m0.NodeID != nil:
		if sofar.Contains(*m0.NodeID) {
			return findQuorumHelper(threshold-1, members[1:], msgs, pred, sofar)
		}
		if msg, ok := msgs[*m0.NodeID]; ok && pred.test(msg) {
			sofar2, pred2 := findQuorumHelper(msg.Q.Threshold, msg.Q.Members, msgs, pred.next(), sofar.Add(*m0.NodeID))
			if len(sofar2) > 0 {
				return findQuorumHelper(threshold-1, members[1:], msgs, pred2, sofar2)
			}
		}

	case m0.QSet != nil:
		sofar2, pred2 := findQuorumHelper(m0.QSet.Threshold, m0.QSet.Members, msgs, pred, sofar)
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
	slicesHelper(q.Threshold, q.Members, f, nil)
}

// t > 0
func slicesHelper(t int, members []QSetMember, f func(NodeIDSet) bool, sofar NodeIDSet) {
	if t > len(members) {
		return
	}

	if t == 1 {
		for _, m := range members {
			switch {
			case m.NodeID != nil:
				if !f(sofar.Add(*m.NodeID)) {
					return
				}

			case m.QSet != nil:
				slicesHelper(m.QSet.Threshold, m.QSet.Members, f, sofar)
			}
		}
		return
	}

	m0 := members[0]
	switch {
	case m0.NodeID != nil:
		slicesHelper(t-1, members[1:], f, append(sofar, *m0.NodeID))

	case m0.QSet != nil:
		slicesHelper(
			m0.QSet.Threshold,
			m0.QSet.Members,
			func(n NodeIDSet) bool {
				slicesHelper(t-1, members[1:], f, sofar.Union(n))
				return true
			},
			sofar,
		)
	}
	slicesHelper(t, members[1:], f, sofar)
}

package scp

// This file contains functions for finding "blocking sets" and
// "quorums" that satisfy a given predicate.
//
// Each node specifies one or more "quorum slices." Each quorum slice
// is a set of trusted peer nodes. Each quorum slice conceptually
// includes the node itself, though in this implementation that is not
// explicit.
//
// A quorum slice is not necessarily a quorum in itself. A peer in a
// quorum slice may have a dependency on a third-party node, as may
// that node, and so on. A quorum (with respect to a given node) is
// thus the transitive closure over any of its quorum slices. A node
// may have many different quorums, and they may overlap one another.
//
// Every protocol message includes the sending node's set of quorum
// slices. Every node saves the latest message seen from a given
// node. If enough messages have been seen, it is possible for a node
// to know the complete membership of one or more quorums.
//
// A "blocking set" is related to the idea of a quorum, but is
// simpler. It's any set of peers among a node's quorum slices that
// blocks the possibility of a quorum. A blocking set satisfying
// statement X precludes the possibility of a quorum satisfying !X.  A
// single peer from each of a node's quorum slices is sufficient to
// form a blocking set.

func (s *Slot) findBlockingSetOrQuorum(pred predicate) []NodeID {
	nodeIDs := s.findBlockingSet(pred)
	if len(nodeIDs) > 0 {
		return nodeIDs
	}
	return s.findQuorum(pred)
}

// Checks that at least one node in each quorum slice satisfies pred.
func (s *Slot) findBlockingSet(pred predicate) []NodeID {
	return s.findBlockingSetHelper(s.V.Q, nil, pred)
}

func (s *Slot) findBlockingSetHelper(slices [][]NodeID, ids []NodeID, pred predicate) []NodeID {
	if len(slices) == 0 {
		return ids
	}
	for _, nodeID := range slices[0] {
		if env, ok := s.M[nodeID]; ok && pred.test(env) {
			nextPred := pred.next()
			res := s.findBlockingSetHelper(slices[1:], append(ids, nodeID), nextPred)
			if len(res) > 0 {
				return res
			}
		}
	}
	return nil
}

// findQuorum finds a quorum containing the slot's node in which every
// node satisfies the given predicate.
func (s *Slot) findQuorum(pred predicate) []NodeID {
	m := make(map[NodeID]struct{})
	m[s.V.ID] = struct{}{}
	m, _ = s.findNodeQuorum(s.V.ID, s.V.Q, pred, m)
	if len(m) == 0 {
		return nil
	}
	result := make([]NodeID, 0, len(m))
	for n := range m {
		result = append(result, n)
	}
	return result
}

// findNodeQuorum is a helper function for findQuorum. It checks that
// the node has at least one slice whose members (and the transitive
// closure over them) all satisfy the given predicate.
func (s *Slot) findNodeQuorum(nodeID NodeID, q [][]NodeID, pred predicate, m map[NodeID]struct{}) (map[NodeID]struct{}, predicate) {
	for _, slice := range q {
		// s.V.Logf("** findNodeQuorum(%s), slice %s", nodeID, slice)
		m2, nextPred := s.findSliceQuorum(slice, pred, m)
		if len(m2) > 0 {
			// s.V.Logf("** findNodeQuorum(%s), slice %s: success", nodeID, slice)
			return m2, nextPred
		}
	}
	// s.V.Logf("** findNodeQuorum(%s): failure", nodeID)
	return nil, pred
}

// findSliceQuorum is a helper function for findNodeQuorum. It checks
// whether every node in a given quorum slice (and the transitive
// closure over them) satisfies the given predicate.
func (s *Slot) findSliceQuorum(slice []NodeID, pred predicate, m map[NodeID]struct{}) (map[NodeID]struct{}, predicate) {
	var newNodeIDs []NodeID
	for _, nodeID := range slice {
		if _, ok := m[nodeID]; !ok {
			newNodeIDs = append(newNodeIDs, nodeID)
		}
	}
	if len(newNodeIDs) == 0 {
		// s.V.Logf("** findSliceQuorum: no new nodes, success")
		return m, pred
	}
	// s.V.Logf("** findSliceQuorum: new nodes %s", newNodeIDs)
	origPred := pred
	for _, nodeID := range newNodeIDs {
		if env, ok := s.M[nodeID]; !ok || !pred.test(env) {
			// s.V.Logf("** findSliceQuorum: failed on %s", nodeID)
			return nil, origPred
		}
		pred = pred.next()
	}
	m2 := make(map[NodeID]struct{})
	for nodeID := range m {
		m2[nodeID] = struct{}{}
	}
	for _, nodeID := range newNodeIDs {
		m2[nodeID] = struct{}{}
	}
	for _, nodeID := range newNodeIDs {
		// s.V.Logf("** findSliceQuorum: transitive call to findNodeQuorum")
		env := s.M[nodeID]
		m2, pred = s.findNodeQuorum(nodeID, env.Q, pred, m2)
		if len(m2) == 0 {
			// s.V.Logf("** findSliceQuorum: transitive call to findNodeQuorum failed")
			return nil, origPred
		}
	}
	// s.V.Logf("** findSliceQuorum: success")
	return m2, pred
}

// Abstract predicate. Concrete types below.
type predicate interface {
	test(*Env) bool

	// next allows a predicate to update itself after each successful
	// call to test, by returning a modified copy of itself for the next
	// call. When findBlockingSet or findQuorum needs to backtrack, they
	// also unwind to earlier versions of the predicate.
	next() predicate
}

// This is a simple function predicate. It does not change from one
// call to the next.
type fpred func(*Env) bool

func (f fpred) test(env *Env) bool {
	return f(env)
}

func (f fpred) next() predicate {
	return f
}

// minMaxPred is a predicate that can narrow a set of min/max bounds
// as it traverses nodes.
type minMaxPred struct {
	min, max           int
	nextMin, nextMax   int
	finalMin, finalMax *int
	testfn             func(env *Env, min, max int) (bool, int, int)
}

func (p *minMaxPred) test(env *Env) bool {
	p.nextMin, p.nextMax = p.min, p.max
	if p.min > p.max {
		return false
	}
	res, min, max := p.testfn(env, p.min, p.max)
	if !res {
		return false
	}
	p.nextMin, p.nextMax = min, max
	return true
}

func (p *minMaxPred) next() predicate {
	if p.finalMin != nil {
		*p.finalMin = p.nextMin
	}
	if p.finalMax != nil {
		*p.finalMax = p.nextMax
	}
	return &minMaxPred{
		min:      p.nextMin,
		max:      p.nextMax,
		finalMin: p.finalMin,
		finalMax: p.finalMax,
		testfn:   p.testfn,
	}
}

package scp

// Predicates are passed to the functions herein as interface{}
// values. Their concrete types must be either func(*Env)bool or
// predicate. A predicate is able to supply new, modified predicates
// as these functions explore the space of candidate nodes, with
// unwinding.

type predicate interface {
	test(*Env) bool
	next() predicate
}

func doTest(env *Env, p interface{}) bool {
	switch p := p.(type) {
	case func(*Env) bool:
		return p(env)
	case predicate:
		return p.test(env)
	}
	panic(fmt.Errorf("got %T, want func(*Env)bool or predicate", p))
}

func getNext(p interface{}) interface{} {
	switch p := p.(type) {
	case func(*Env) bool:
		return p
	case predicate:
		return p.next()
	}
	panic(fmt.Errorf("got %T, want func(*Env)bool or predicate", p))
}

func (s *Slot) findBlockingSetOrQuorum(pred interface{}) []NodeID {
	nodeIDs := s.findBlockingSet(pred)
	if len(nodeIDs) > 0 {
		return nodeIDs
	}
	return s.findQuorum(pred)
}

// Checks that at least one node in each quorum slice satisfies pred.
func (s *Slot) findBlockingSet(pred interface{}) []NodeID {
	return s.findBlockingSetHelper(s.V.Q, nil, pred)
}

func (s *Slot) findBlockingSetHelper(slices [][]NodeID, ids []NodeID, pred interface{}) []NodeID {
	if len(slices) == 0 {
		return ids
	}
	for _, nodeID := range slices[0] {
		if env, ok := s.M[nodeID]; ok && doTest(env, pred) {
			nextPred := getNext(pred)
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
func (s *Slot) findQuorum(pred interface{}) []NodeID {
	m := make(map[NodeID]struct{})
	m = s.findNodeQuorum(s.V.ID, s.V.Q, pred, m)
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
func (s *Slot) findNodeQuorum(nodeID NodeID, q QSet, pred interface{}, m map[NodeID]struct{}) map[NodeID]struct{} {
	for _, slice := range q {
		m2 := s.findSliceQuorum(slice, pred, m)
		if len(m2) > 0 {
			return m2
		}
	}
	return nil
}

// findSliceQuorum is a helper function for findNodeQuorum. It checks
// whether every node in a given quorum slice (and the transitive
// closure over them) satisfies the given predicate.
func (s *Slot) findSliceQuorum(slice []NodeID, pred interface{}, m map[NodeID]struct{}) map[NodeID]struct{} {
	var newNodeIDs []NodeID
	for _, nodeID := range slice {
		if _, ok := m[nodeID]; !ok {
			newNodeIDs = append(newNodeIDs, nodeID)
		}
	}
	if len(newNodeIDs) == 0 {
		return m
	}
	for _, nodeID := range newNodeIDs {
		if env, ok := s.M[nodeID]; !ok || !doTest(env, pred) {
			return nil
		}
		pred = getNext(pred)
	}
	m2 := make(map[NodeID]struct{})
	for nodeID := range m {
		m2[nodeID] = struct{}{}
	}
	for _, nodeID := range newNodeIDs {
		m2[nodeID] = struct{}{}
	}
	for _, nodeID := range newNodeIDs {
		env, ok := s.M[nodeID]
		if !ok {
			return nil
		}
		m2 = s.findNodeQuorum(nodeID, env.Q, pred, m2)
		if len(m2) == 0 {
			return nil
		}
	}
	return m2
}

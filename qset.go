package scp

// QSet is a quorum set, a set of quorum slices.
// For compactness, the QSet in a node V, or in a message from a node
// V, does not specify V, though V is understood to be in every slice.
type QSet [][]NodeID

func (q QSet) Each(f func([]NodeID)) {
	for _, slice := range q {
		f(slice)
	}
}

func (q QSet) Size() int {
	return len(q)
}

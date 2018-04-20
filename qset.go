package scp

// QSet is a quorum set, a set of quorum slices.
// For compactness, the QSet in a node V, or in a message from a node
// V, does not specify V, though V is understood to be in every slice.
type QSet [][]NodeID

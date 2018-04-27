package main

import "sort"

// The concrete type for scp.Value. This network votes on hex-encoded
// block ID strings. When a node needs to know the contents of a
// block, it can inquire via RPC.
type valtype string

func (v valtype) Less(other valtype) bool {
	return v.Less(other)
}

func (v valtype) Combine(other valtype) valtype {
	blockMapMu.Lock()
	var (
		// xxx what to do when we don't have the actual blocks?
		b1 = blockMap[v]
		b2 = blockMap[other]
	)
	blockMapMu.Unlock()

	txs := b1.Transactions
	txs = append(txs, b2.Transactions...)
	// xxx topo-sort txs, commutatively (so v.Combine(other) == other.Combine(v))
}

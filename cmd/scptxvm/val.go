package main

import (
	"encoding/hex"
	"sort"

	"github.com/chain/txvm/protocol/bc"
)

// The concrete type for scp.Value. This network votes on block
// IDs. When a node needs to know the contents of a block, it can
// inquire via RPC.
type valtype bc.Hash

func (v valtype) Less(other valtype) bool {
	if v.V0 < other.V0 {
		return true
	}
	if v.V0 > other.V0 {
		return false
	}
	if v.V1 < other.V1 {
		return true
	}
	if v.V1 > other.V1 {
		return false
	}
	if v.V2 < other.V2 {
		return true
	}
	if v.V2 > other.V2 {
		return false
	}
	return v.V3 < other.V3
}

func (v valtype) String() string {
	return hex.EncodeToString(bc.Hash(v).Bytes())
}

func (v valtype) Combine(other valtype) valtype {
	blockMapMu.Lock()
	var (
		// xxx what to do when we don't have the actual blocks?
		b1 = blockMap[v]
		b2 = blockMap[other]
	)
	blockMapMu.Unlock()

	// xxx to combine, blocks must have the same Height, PreviousBlockId,

	txs := b1.Transactions
	txs = append(txs, b2.Transactions...)
	sort.Slice(txs, func(i, j int) bool {
		if xxx /* txs[i] outputs overlap txs[j] inputs */ {
			return true
		}
		if xxx /* txs[j] outputs overlap txs[i] inputs */ {
			return false
		}
		return txs[i].ID < txs[j].ID
	})

	// Eliminate duplicates. There should be no more than two of any
	// given txid, but this code handles any number of duplicates
	// anyway.
	var (
		n = 0
		d = 1
	)
	for n+d < len(txs) { // xxx double-check the logic in this loop
		if txs[n].ID == txs[n+d].ID {
			d++
		} else {
			if d > 1 {
				txs[n+1] = txs[n+d]
				n++
			}
		}
	}
	txs = txs[:n]

	// xxx create a new block
	// xxx if not possible to create a new block, choose one based on
	// blockID and slotID (which is the block height)
}

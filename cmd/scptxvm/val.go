package main

import (
	"encoding/hex"
	"sort"

	"github.com/bobg/scp"
	"github.com/chain/txvm/protocol/bc"
)

// The concrete type for scp.Value. This network votes on block
// IDs. When a node needs to know the contents of a block, it can
// inquire via RPC.
type valtype bc.Hash

func (v valtype) Less(otherval scp.Value) bool {
	other := otherval.(valtype)
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

func (v valtype) Bytes() []byte {
	return bc.Hash(v).Bytes()
}

func (v valtype) String() string {
	return hex.EncodeToString(bc.Hash(v).Bytes())
}

func (v valtype) Combine(otherval scp.Value, slotID scp.SlotID) scp.Value {
	other := otherval.(valtype)
	if other.Less(v) {
		return other.Combine(v, slotID)
	}
	if !v.Less(other) {
		// v == other
		return v
	}

	b1, err := getBlock(int(slotID), bc.Hash(v))
	if err != nil {
		panic(err) // xxx is this OK?
	}
	b2, err := getBlock(int(slotID), bc.Hash(other))
	if err != nil {
		panic(err) // xxx
	}

	txs := b1.Transactions
	txs = append(txs, b2.Transactions...)
	sort.Slice(txs, func(i, j int) bool {
		s := make(map[bc.Hash]struct{})
		for _, out := range txs[i].Outputs {
			s[out.ID] = struct{}{}
		}
		for _, in := range txs[j].Inputs {
			if _, ok := s[in.ID]; ok {
				return true
			}
		}

		s = make(map[bc.Hash]struct{})
		for _, out := range txs[j].Outputs {
			s[out.ID] = struct{}{}
		}
		for _, in := range txs[i].Inputs {
			if _, ok := s[in.ID]; ok {
				return false
			}
		}

		return valtype(txs[i].ID).Less(valtype(txs[j].ID))
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
	cmtxs := make([]*bc.CommitmentsTx, 0, len(txs))
	for _, tx := range txs {
		cmtxs = append(cmtxs, bc.NewCommitmentsTx(tx))
	}

	// Use the earlier timestamp.
	timestampMS := b1.TimestampMs
	if b2.TimestampMs < timestampMS {
		timestampMS = b2.TimestampMs
	}
	snapshot := chain.State()
	ublock, _, err := chain.GenerateBlock(bgctx, snapshot, timestampMS, cmtxs)
	if err != nil {
		// Cannot make a block from the combined set of txs. Choose one of
		// the input blocks as the winner.
		if slotID%2 == 0 {
			return v
		}
		return other
	}

	block, err := bc.SignBlock(ublock, snapshot.Header, nil)
	if err != nil {
		panic(err)
	}

	err = storeBlock(block)
	if err != nil {
		panic(err)
	}

	return valtype(block.Hash())
}

func (v valtype) IsNil() bool {
	h := bc.Hash(v)
	return h.IsZero()
}

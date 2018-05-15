package main

import (
	"context"
	"time"

	"github.com/bobg/scp"
	"github.com/chain/txvm/protocol/bc"
)

func nominate(ctx context.Context) {
	defer wg.Done()

	txpool := make(map[bc.Hash]*bc.Tx)

	doNom := func() error {
		if len(txpool) == 0 {
			return nil
		}

		txs := make([]*bc.CommitmentsTx, 0, len(txpool))
		for _, tx := range txpool {
			txs = append(txs, bc.NewCommitmentsTx(tx))
		}

		timestampMS := bc.Millis(time.Now())
		snapshot := chain.State()
		if snapshot.Header.TimestampMs > timestampMS {
			timestampMS = snapshot.Header.TimestampMs + 1 // xxx sleep until this time? error out?
		}

		block, _, err := chain.GenerateBlock(ctx, snapshot, timestampMS, txs)
		if err != nil {
			return err
		}
		// xxx figure out which txs GenerateBlock removed as invalid, and remove them from txpool

		err = storeBlock(block)
		if err != nil {
			return err
		}
		n := &scp.NomTopic{
			X: scp.ValueSet{valtype(block.Hash())},
		}
		node.Logf("nominating block %x (%d tx(s)) at height %d", block.Hash().Bytes(), len(block.Transactions), block.Height)
		msg := scp.NewMsg(node.ID, scp.SlotID(block.Height), node.Q, n) // xxx slotID is 32 bits, block height is 64
		node.Handle(msg)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			node.Logf("context canceled, exiting nominate loop")
			return

		case item := <-nomChan:
			switch item := item.(type) {
			case *bc.Tx:
				if _, ok := txpool[item.ID]; ok {
					// tx is already in the pool
					continue
				}
				node.Logf("adding tx %x to the pool", item.ID.Bytes())
				txpool[item.ID] = item // xxx need to persist this
				err := doNom()
				if err != nil {
					panic(err) // xxx
				}

			case scp.SlotID:
				err := doNom()
				if err != nil {
					panic(err) // xxx
				}

			case *bc.Block:
				// Remove published and conflicting txs from txpool.
				spent := make(map[bc.Hash]struct{})
				for _, tx := range item.Transactions {
					for _, inp := range tx.Inputs {
						spent[inp.ID] = struct{}{}
					}
					// Published tx.
					node.Logf("removing published tx %x from the pool", tx.ID.Bytes())
					delete(txpool, tx.ID)
				}
				for id, tx := range txpool {
					for _, inp := range tx.Inputs {
						if _, ok := spent[inp.ID]; ok {
							// Conflicting tx.
							node.Logf("removing conflicting tx %x from the pool", id.Bytes())
							delete(txpool, id)
							break
						}
					}
				}
			}
		}
	}
}

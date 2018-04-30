package main

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/state"
)

// Implements protocol.Store from github.com/chain/txvm
type pstore struct {
	height   uint64
	dir      string
	snapshot *state.Snapshot
}

func (s pstore) Height() (uint64, error) {
	return s.height
}

func (s pstore) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	filename, err := s.findSavedBlockFilename(height)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var block bc.Block
	err = block.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (s pstore) LatestSnapshot(ctx context.Context) (*state.Snapshot, error) {
	dir := s.snapshotDir()
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// xxx find the highest snapshot
	filename = path.Join(dir, filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var snapshot state.Snapshot
	err = snapshot.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s pstore) SaveBlock(ctx context.Context, block *bc.Block) error {
	filename := s.savedBlockFilename(block)
	b, err := block.Bytes()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}

func (s pstore) FinalizeHeight(ctx context.Context, height uint64) error {
	return nil
}

func (s pstore) SaveSnapshot(ctx context.Context, snapshot *state.Snapshot) error {
	filename := s.snapshotFilename(snapshot)
	b, err := snapshot.Bytes()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}

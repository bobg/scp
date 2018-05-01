package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"

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

func (s pstore) GetBlock(_ context.Context, height uint64) (*bc.Block, error) {
	return readBlockFile(path.Join(blockDir(), strconv.Itoa(height)))
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
	err := storeBlock(block)
	if err != nil {
		return err
	}
	oldName := blockFilename(block.Height, block.Hash())
	newName := path.Join(blockDir(), strconv.Itoa(block.Height))
	err := os.Link(oldName, newName)
	if os.IsExist(err) {
		return nil
	}
	return err
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

func blockFilename(height int, id bc.Hash) string {
	return path.Join(blockDir(), fmt.Sprintf("%d-%x", height, id.Bytes()))
}

func getBlock(height int, id bc.Hash) (*bc.Block, err) {
	return readBlockFile(blockFilename(height, id))
}

func readBlockFile(filename string) (*bc.Block, err) {
	bits, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var block bc.Block
	err = block.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

var storeBlockMu sync.Mutex

func storeBlock(block *bc.Block) error {
	storeBlockMu.Lock()
	defer storeBlockMu.Unlock()

	filename := blockFilename(block.Height, block.Hash())
	_, err := os.Stat(filename)
	if err == nil {
		// File exists already.
		return nil
	}
	if !os.IsNotExist(err) {
		// Problem is other than file-doesn't-exist.
		return err
	}
	bits, err := block.Bytes()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, block.Bytes(), 0644)
}

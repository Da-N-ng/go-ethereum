// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/light"
)

// func TestCheckpointSyncingLes1(t *testing.T) { testCheckpointSyncing(t, 1) }
func TestCheckpointSyncingLes2(t *testing.T) { testCheckpointSyncing(t, 2) }

func testCheckpointSyncing(t *testing.T, protocol int) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			bs, _, _ := bIndexer.Sections()
			bts, _, _ := btIndexer.Sections()
			if cs >= (config.PairChtSize)/config.ChtSize && bs >= config.PairChtSize/config.BloomSize &&
				bts >= config.PairChtSize/config.BloomTrieSize {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Generate 512+4 blocks (totally 8 CHT sections)
	server, client, tearDown := newClientServerEnv(t, int(config.PairChtSize+config.ChtConfirms), protocol, waitIndexers, false)
	defer tearDown()

	// Register checkpoint 0 into the contract at block (512+4)+1
	bts, _, head := server.bloomTrieIndexer.Sections()
	chtRoot := light.GetChtRoot(server.db, 7, head)
	btRoot := light.GetBloomTrieRoot(server.db, bts-1, head)

	cp := &light.TrustedCheckpoint{
		SectionIndex: 0,
		SectionHead:  head,
		CHTRoot:      chtRoot,
		BloomRoot:    btRoot,
	}
	if _, err := server.pm.reg.contract.SetCheckpoint(signerKey, big.NewInt(int64(cp.SectionIndex)), cp.Hash().Bytes()); err != nil {
		t.Error("register checkpoint failed", err)
	}
	server.backend.Commit()
	server.backend.ShiftBlocks(1)

	// Wait for the checkpoint registration
	for {
		hash, _, err := server.pm.reg.contract.Contract().GetCheckpoint(nil, big.NewInt(0))
		if err != nil || hash == [32]byte{} {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}

	done := make(chan error)
	client.pm.reg.SyncDoneHook = func() {
		header := client.pm.blockchain.CurrentHeader()
		if header.Number.Uint64() == config.PairChtSize+config.ChtConfirms+2 {
			done <- nil
		} else {
			done <- errors.New("blockchain mismatch")
		}
	}

	// Create connected peer pair.
	peer, err1, lPeer, err2 := newTestPeerPair("peer", protocol, server.pm, client.pm)
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 2 handshake error: %v", err)
	}
	server.rPeer, client.rPeer = peer, lPeer

	select {
	case err := <-done:
		if err != nil {
			t.Error("sync failed", err)
		}
		return
	case <-time.NewTimer(10 * time.Second).C:
		t.Error("checkpoint syncing timeout")
	}
}

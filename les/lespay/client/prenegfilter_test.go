// Copyright 2020 The go-ethereum Authors
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

package client

import (
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

const (
	pfTestNegative = 4000
	pfTestPositive = 1000
	pfTestTimeout  = 5000
	pfTestTotal    = pfTestNegative + pfTestPositive + pfTestTimeout
)

func TestPreNegFilter(t *testing.T) {
	clock := &mclock.Simulated{}
	ns := nodestate.NewNodeStateMachine(nil, nil, clock, testSetup)
	nodes := make([]*enode.Node, pfTestTotal)
	for i := range nodes {
		nodes[i] = enode.SignNull(&enr.Record{}, testNodeID(i))
	}
	queryResult := make([]int, pfTestTotal)
	for c := pfTestPositive; c > 0; c-- {
		for {
			i := rand.Intn(pfTestTotal - 5)
			if queryResult[i] == 0 {
				queryResult[i] = 1
				break
			}
		}
	}
	for c := pfTestTimeout; c > 0; c-- {
		for {
			i := rand.Intn(pfTestTotal - 5)
			if queryResult[i] == 0 {
				queryResult[i] = 2
				break
			}
		}
	}

	var timeout uint32
	testQuery := func(node *enode.Node, result func(canDial bool)) func() {
		idx := testNodeIndex(node.ID())
		switch queryResult[idx] {
		case 0:
			return func() { result(false) } // negative answer
		case 1:
			return func() { result(true) } // positive answer
		case 2:
			return func() {
				clock.Run(5 * time.Second)
				atomic.AddUint32(&timeout, 1)
			} // timeout
		}
		return nil
	}
	pf := NewPreNegFilter(ns, enode.IterNodes(nodes), testQuery, sfTest1, sfTest2, 5, time.Second*5, time.Second*10)
	pfr := enode.Filter(pf, func(node *enode.Node) bool {
		// remove canDial flag so that filter can keep querying
		ns.SetState(node, nodestate.Flags{}, sfTest2, 0)
		return true
	})
	ns.Start()

	l := len(enode.ReadNodes(pfr, pfTestTotal))
	if l != pfTestPositive {
		t.Errorf("Wrong number of returned result (got %d, expected %d)", l, pfTestPositive)
	}
	if atomic.LoadUint32(&timeout) != pfTestTimeout {
		t.Errorf("Wrong amount of timeout query number(got %v, expected %v)", timeout, pfTestTimeout)
	}
}
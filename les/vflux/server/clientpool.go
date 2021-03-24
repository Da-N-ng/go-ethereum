// Copyright 2019 The go-ethereum Authors
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

package server

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/les/vflux"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	serverSetup = &nodestate.Setup{}
	clientField = serverSetup.NewField("client", reflect.TypeOf(clientPeerInstance{}))
	btSetup     = newBalanceTrackerSetup(serverSetup)
	ppSetup     = newPriorityPoolSetup(serverSetup)
)

var (
	ErrNotConnected    = errors.New("client not connected")
	ErrNoPriority      = errors.New("priority too low to raise capacity")
	ErrCantFindMaximum = errors.New("Unable to find maximum allowed capacity")
)

func init() {
	btSetup.connect(clientField, ppSetup.capacityField)
	ppSetup.connect(btSetup.balanceField, btSetup.updateFlag) // nodeBalance implements nodePriority
}

// ClientPool implements a client database that assigns a priority to each client
// based on a positive and negative balance. Positive balance is externally assigned
// to prioritized clients and is decreased with connection time and processed
// requests (unless the price factors are zero). If the positive balance is zero
// then negative balance is accumulated.
//
// Balance tracking and priority calculation for connected clients is done by
// balanceTracker. activeQueue ensures that clients with the lowest positive or
// highest negative balance get evicted when the total capacity allowance is full
// and new clients with a better balance want to connect.
//
// Already connected nodes receive a small bias in their favor in order to avoid
// accepting and instantly kicking out clients. In theory, we try to ensure that
// each client can have several minutes of connection time.
//
// Balances of disconnected clients are stored in nodeDB including positive balance
// and negative banalce. Boeth positive balance and negative balance will decrease
// exponentially. If the balance is low enough, then the record will be dropped.
type ClientPool struct {
	*priorityPool
	*balanceTracker
	clock  mclock.Clock
	closed bool
	ns     *nodestate.NodeStateMachine
	synced func() bool

	lock                                 sync.RWMutex
	defaultPosFactors, defaultNegFactors PriceFactors
	connectedBias                        time.Duration

	minCap     uint64      // the minimal capacity value allowed for any client
	capReqNode *enode.Node // node that is requesting capacity change; only used inside NSM operation

	capacityQueryZeroMeter, capacityQueryNonZeroMeter metrics.Meter
}

// clientPeer represents a peer in the client pool. None of the callbacks should block.
type clientPeer interface {
	Node() *enode.Node
	FreeClientId() string                         // unique id for non-priority clients (typically a prefix of the network address)
	InactiveTimeout() time.Duration               // disconnection timeout for inactive non-priority peers
	UpdateCapacity(newCap uint64, requested bool) // signals a capacity update (requested is true if it is a result of a SetCapacity call on the given peer
	Disconnect()                                  // initiates disconnection (Unregister should always be called)
}

type clientPeerInstance struct{ clientPeer } // the NodeStateMachine type system needs this wrapper

// NewClientPool creates a new client pool
func NewClientPool(balanceDb ethdb.KeyValueStore, minCap uint64, connectedBias time.Duration, clock mclock.Clock, synced func() bool) *ClientPool {
	ns := nodestate.NewNodeStateMachine(nil, nil, clock, serverSetup)
	cp := &ClientPool{
		ns:             ns,
		balanceTracker: newBalanceTracker(ns, btSetup, balanceDb, clock, &utils.Expirer{}, &utils.Expirer{}),
		priorityPool:   newPriorityPool(ns, ppSetup, clock, minCap, connectedBias, 4),
		clock:          clock,
		minCap:         minCap,
		connectedBias:  connectedBias,
		synced:         synced,
	}

	ns.SubscribeState(nodestate.MergeFlags(ppSetup.activeFlag, ppSetup.inactiveFlag, btSetup.priorityFlag), func(node *enode.Node, oldState, newState nodestate.Flags) {
		if newState.Equals(ppSetup.inactiveFlag) {
			// set timeout for non-priority inactive client
			var timeout time.Duration
			if c, ok := ns.GetField(node, clientField).(clientPeer); ok {
				timeout = c.InactiveTimeout()
			}
			if timeout > 0 {
				ns.AddTimeout(node, ppSetup.inactiveFlag, timeout)
			} else {
				// Note: if capacity is immediately available then priorityPool will set the active
				// flag simultaneously with removing the inactive flag and therefore this will not
				// initiate disconnection
				ns.SetStateSub(node, nodestate.Flags{}, ppSetup.inactiveFlag, 0)
			}
		}
		if oldState.Equals(ppSetup.inactiveFlag) && newState.Equals(ppSetup.inactiveFlag.Or(btSetup.priorityFlag)) {
			ns.SetStateSub(node, ppSetup.inactiveFlag, nodestate.Flags{}, 0) // priority gained; remove timeout
		}
		if newState.Equals(ppSetup.activeFlag) {
			// active with no priority; limit capacity to minCap
			cap, _ := ns.GetField(node, ppSetup.capacityField).(uint64)
			if cap > minCap {
				cp.requestCapacity(node, minCap, 0, true)
			}
		}
		if newState.Equals(nodestate.Flags{}) {
			if c, ok := ns.GetField(node, clientField).(clientPeer); ok {
				c.Disconnect()
			}
		}
	})

	ns.SubscribeField(btSetup.balanceField, func(node *enode.Node, state nodestate.Flags, oldValue, newValue interface{}) {
		if newValue != nil {
			ns.SetStateSub(node, ppSetup.inactiveFlag, nodestate.Flags{}, 0)
			cp.lock.RLock()
			newValue.(*nodeBalance).SetPriceFactors(cp.defaultPosFactors, cp.defaultNegFactors)
			cp.lock.RUnlock()
		}
	})

	ns.SubscribeField(ppSetup.capacityField, func(node *enode.Node, state nodestate.Flags, oldValue, newValue interface{}) {
		if c, ok := ns.GetField(node, clientField).(clientPeer); ok {
			newCap, _ := newValue.(uint64)
			c.UpdateCapacity(newCap, node == cp.capReqNode)
		}
	})
	return cp
}

// AddMetrics adds metrics to the client pool. Should be called before Start().
func (cp *ClientPool) AddMetrics(totalConnectedGauge metrics.Gauge,
	clientConnectedMeter, clientDisconnectedMeter, clientActivatedMeter, clientDeactivatedMeter,
	capacityQueryZeroMeter, capacityQueryNonZeroMeter metrics.Meter) {

	cp.ns.SubscribeState(nodestate.MergeFlags(ppSetup.activeFlag, ppSetup.inactiveFlag), func(node *enode.Node, oldState, newState nodestate.Flags) {
		if oldState.IsEmpty() && !newState.IsEmpty() {
			clientConnectedMeter.Mark(1)
		}
		if !oldState.IsEmpty() && newState.IsEmpty() {
			clientDisconnectedMeter.Mark(1)
		}
		if oldState.HasNone(ppSetup.activeFlag) && oldState.HasAll(ppSetup.activeFlag) {
			clientActivatedMeter.Mark(1)
		}
		if oldState.HasAll(ppSetup.activeFlag) && oldState.HasNone(ppSetup.activeFlag) {
			clientDeactivatedMeter.Mark(1)
		}
		_, connected := cp.Active()
		totalConnectedGauge.Update(int64(connected))
	})
	cp.capacityQueryZeroMeter = capacityQueryZeroMeter
	cp.capacityQueryNonZeroMeter = capacityQueryNonZeroMeter
}

// Start starts the client pool. Should be called before Register/Unregister.
func (cp *ClientPool) Start() {
	cp.ns.Start()
}

// Stop shuts the client pool down. The clientPeer interface callbacks will not be called
// after Stop. Register calls will return nil.
func (cp *ClientPool) Stop() {
	cp.balanceTracker.stop()
	cp.ns.Stop()
}

// Register registers the peer into the client pool. If the peer has insufficient
// priority and remains inactive for longer than the allowed timeout then it will be
// disconnected by calling the Disconnect function of the clientPeer interface.
func (cp *ClientPool) Register(peer clientPeer) ConnectedBalance {
	cp.ns.SetField(peer.Node(), clientField, clientPeerInstance{peer})
	balance, _ := cp.ns.GetField(peer.Node(), btSetup.balanceField).(*nodeBalance)
	return balance
}

// Unregister removes the peer from the client pool
func (cp *ClientPool) Unregister(peer clientPeer) {
	cp.ns.SetField(peer.Node(), clientField, nil)
}

// SetDefaultFactors sets the default price factors applied to subsequently connected clients
func (cp *ClientPool) SetDefaultFactors(posFactors, negFactors PriceFactors) {
	cp.lock.Lock()
	cp.defaultPosFactors = posFactors
	cp.defaultNegFactors = negFactors
	cp.lock.Unlock()
}

// setConnectedBias sets the connection bias, which is applied to already connected clients
// So that already connected client won't be kicked out very soon and we can ensure all
// connected clients can have enough time to request or sync some data.
func (cp *ClientPool) SetConnectedBias(bias time.Duration) {
	cp.lock.Lock()
	cp.connectedBias = bias
	cp.setActiveBias(bias)
	cp.lock.Unlock()
}

// SetCapacity sets the assigned capacity of a connected client
func (cp *ClientPool) SetCapacity(node *enode.Node, reqCap uint64, bias time.Duration, requested bool) (capacity uint64, err error) {
	cp.lock.RLock()
	if cp.connectedBias > bias {
		bias = cp.connectedBias
	}
	cp.lock.RUnlock()

	cp.ns.Operation(func() {
		balance, _ := cp.ns.GetField(node, btSetup.balanceField).(*nodeBalance)
		if balance == nil {
			err = ErrNotConnected
			return
		}
		capacity, _ = cp.ns.GetField(node, ppSetup.capacityField).(uint64)
		if capacity == 0 {
			// if the client is inactive then it has insufficient priority for the minimal capacity
			// (will be activated automatically with minCap when possible)
			return
		}
		if reqCap < cp.minCap {
			// can't request less than minCap; switching between 0 (inactive state) and minCap is
			// performed by the server automatically as soon as necessary/possible
			reqCap = cp.minCap
		}
		if reqCap > cp.minCap {
			if cp.ns.GetState(node).HasNone(btSetup.priorityFlag) && reqCap > cp.minCap {
				err = ErrNoPriority
				return
			}
		}
		if reqCap == capacity {
			return
		}
		curveBias := bias
		if requested {
			// mark the requested node so that the UpdateCapacity callback can signal
			// whether the update is the direct result of a SetCapacity call on the given node
			cp.capReqNode = node
			defer func() {
				cp.capReqNode = nil
			}()
		}

		// estimate maximum available capacity at the current priority level and request
		// the estimated amount; allow a limited number of retries because individual
		// balances can change between the estimation and the request
		for count := 0; count < 20; count++ {
			// apply a small extra bias to ensure that the request won't fail because of rounding errors
			curveBias += time.Second * 10
			tryCap := reqCap
			if reqCap > capacity {
				curve := cp.getCapacityCurve().exclude(node.ID())
				tryCap = curve.maxCapacity(func(capacity uint64) int64 {
					return balance.estimatePriority(capacity, 0, 0, curveBias, false)
				})
				if tryCap <= capacity {
					return
				}
				if tryCap > reqCap {
					tryCap = reqCap
				}
			}
			if _, allowed := cp.requestCapacity(node, tryCap, bias, true); allowed {
				capacity = tryCap
				return
			}
		}
		// we should be able to find the maximum allowed capacity in a few iterations
		log.Error("Unable to find maximum allowed capacity")
		err = ErrCantFindMaximum
	})
	return
}

// serveCapQuery serves a vflux capacity query. It receives multiple token amount values
// and a bias time value. For each given token amount it calculates the maximum achievable
// capacity in case the amount is added to the balance.
func (cp *ClientPool) serveCapQuery(id enode.ID, freeID string, data []byte) []byte {
	var req vflux.CapacityQueryReq
	if rlp.DecodeBytes(data, &req) != nil {
		return nil
	}
	if l := len(req.AddTokens); l == 0 || l > vflux.CapacityQueryMaxLen {
		return nil
	}
	result := make(vflux.CapacityQueryReply, len(req.AddTokens))
	if !cp.synced() {
		if cp.capacityQueryZeroMeter != nil {
			cp.capacityQueryZeroMeter.Mark(1)
		}
		reply, _ := rlp.EncodeToBytes(&result)
		return reply
	}

	bias := time.Second * time.Duration(req.Bias)
	cp.lock.RLock()
	if cp.connectedBias > bias {
		bias = cp.connectedBias
	}
	cp.lock.RUnlock()

	// use capacityCurve to answer request for multiple newly bought token amounts
	curve := cp.getCapacityCurve().exclude(id)
	cp.BalanceOperation(id, freeID, func(balance AtomicBalanceOperator) {
		pb, _ := balance.GetBalance()
		for i, addTokens := range req.AddTokens {
			add := addTokens.Int64()
			result[i] = curve.maxCapacity(func(capacity uint64) int64 {
				return balance.estimatePriority(capacity, add, 0, bias, false) / int64(capacity)
			})
			if add <= 0 && uint64(-add) >= pb && result[i] > cp.minCap {
				result[i] = cp.minCap
			}
			if result[i] < cp.minCap {
				result[i] = 0
			}
		}
	})
	// add first result to metrics (don't care about priority client multi-queries yet)
	if result[0] == 0 {
		if cp.capacityQueryZeroMeter != nil {
			cp.capacityQueryZeroMeter.Mark(1)
		}
	} else {
		if cp.capacityQueryNonZeroMeter != nil {
			cp.capacityQueryNonZeroMeter.Mark(1)
		}
	}
	reply, _ := rlp.EncodeToBytes(&result)
	return reply
}

// Handle implements Service
func (cp *ClientPool) Handle(id enode.ID, address string, name string, data []byte) []byte {
	switch name {
	case vflux.CapacityQueryName:
		return cp.serveCapQuery(id, address, data)
	default:
		return nil
	}
}
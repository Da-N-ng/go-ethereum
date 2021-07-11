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

package trie

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	memcacheCleanHitMeter   = metrics.NewRegisteredMeter("trie/memcache/clean/hit", nil)
	memcacheCleanMissMeter  = metrics.NewRegisteredMeter("trie/memcache/clean/miss", nil)
	memcacheCleanReadMeter  = metrics.NewRegisteredMeter("trie/memcache/clean/read", nil)
	memcacheCleanWriteMeter = metrics.NewRegisteredMeter("trie/memcache/clean/write", nil)

	memcacheDirtyHitMeter   = metrics.NewRegisteredMeter("trie/memcache/dirty/hit", nil)
	memcacheDirtyMissMeter  = metrics.NewRegisteredMeter("trie/memcache/dirty/miss", nil)
	memcacheDirtyReadMeter  = metrics.NewRegisteredMeter("trie/memcache/dirty/read", nil)
	memcacheDirtyWriteMeter = metrics.NewRegisteredMeter("trie/memcache/dirty/write", nil)

	memcacheFlushTimeTimer  = metrics.NewRegisteredResettingTimer("trie/memcache/flush/time", nil)
	memcacheFlushNodesMeter = metrics.NewRegisteredMeter("trie/memcache/flush/nodes", nil)
	memcacheFlushSizeMeter  = metrics.NewRegisteredMeter("trie/memcache/flush/size", nil)

	memcacheGCTimeTimer  = metrics.NewRegisteredResettingTimer("trie/memcache/gc/time", nil)
	memcacheGCNodesMeter = metrics.NewRegisteredMeter("trie/memcache/gc/nodes", nil)
	memcacheGCSizeMeter  = metrics.NewRegisteredMeter("trie/memcache/gc/size", nil)

	memcacheCommitTimeTimer  = metrics.NewRegisteredResettingTimer("trie/memcache/commit/time", nil)
	memcacheCommitNodesMeter = metrics.NewRegisteredMeter("trie/memcache/commit/nodes", nil)
	memcacheCommitSizeMeter  = metrics.NewRegisteredMeter("trie/memcache/commit/size", nil)
)

const (
	// commitBloomSize is the rough trie node number of each commit operation
	// (and all partial commits). The value can be twisted a bit based on the
	// experience.
	commitBloomSize = 3000_000

	// maxFalsePositiveRate is the maximum acceptable bloom filter false-positive
	// rate to aviod too many useless operations.
	maxFalsePositiveRate = 0.01

	// minBlockConfirms is the minimal block confirms on top for executing pruning.
	minBlockConfirms = 3600
)

// metaRoot is the identifier of the global memcache root that anchors the block
// accounts tries for garbage collection.
const metaRoot = ""

// Database is an intermediate write layer between the trie data structures and
// the disk database. The aim is to accumulate trie writes in-memory and only
// periodically flush a couple tries to disk, garbage collecting the remainder.
//
// Note, the trie Database is **not** thread safe in its mutations, but it **is**
// thread safe in providing individual, independent node access. The rationale
// behind this split design is to provide read access to RPC handlers and sync
// servers even while the trie is executing expensive garbage collection.
type Database struct {
	diskdb  ethdb.KeyValueStore    // Persistent storage for matured trie nodes
	cleans  *fastcache.Cache       // GC friendly memory cache of clean node RLPs
	dirties map[string]*cachedNode // Data and references relationships of dirty trie nodes
	oldest  string                 // Oldest tracked node, flush-list head
	newest  string                 // Newest tracked node, flush-list tail

	preimages map[common.Hash][]byte // Preimages of nodes from the secure trie

	gctime  time.Duration      // Time spent on garbage collection since last commit
	gcnodes uint64             // Nodes garbage collected since last commit
	gcsize  common.StorageSize // Data storage garbage collected since last commit

	flushtime  time.Duration      // Time spent on data flushing since last commit
	flushnodes uint64             // Nodes flushed since last commit
	flushsize  common.StorageSize // Data storage flushed since last commit

	dirtiesSize   common.StorageSize // Storage size of the dirty node cache (exc. metadata)
	childrenSize  common.StorageSize // Storage size of the external children tracking
	preimagesSize common.StorageSize // Storage size of the preimages cache

	pruner *pruner // Pruner for in-disk trie nodes pruning

	lock sync.RWMutex
}

// rawFullNode represents only the useful data content of a full node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawFullNode [17]node

func (n rawFullNode) cache() (hashNode, bool)   { panic("this should never end up in a live trie") }
func (n rawFullNode) fstring(ind string) string { panic("this should never end up in a live trie") }

func (n rawFullNode) EncodeRLP(w io.Writer) error {
	var nodes [17]node

	for i, child := range n {
		if child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

// rawShortNode represents only the useful data content of a short node, with the
// caches and flags stripped out to minimize its data storage. The key of the node
// is in hexary format and needs compact conversion for RLP encoding.
type rawShortNode struct {
	Key []byte
	Val node
}

func (n rawShortNode) cache() (hashNode, bool)   { panic("this should never end up in a live trie") }
func (n rawShortNode) fstring(ind string) string { panic("this should never end up in a live trie") }
func (n rawShortNode) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, shortNode{
		Key: hexToCompact(n.Key),
		Val: n.Val,
	})
}

// cachedNode is all the information we know about a single cached trie node
// in the memory database write layer.
type cachedNode struct {
	node node   // Cached collapsed trie node
	size uint16 // Byte size of the useful cached data

	parents  uint32            // Number of live nodes referencing this one
	children map[string]uint16 // External children referenced by this node

	flushPrev string // Previous node in the flush-list
	flushNext string // Next node in the flush-list
}

// iterateRefs walks the embedded children of the cached node, tracking the
// internal path and invoking the provided callback on all hash nodes.
func (n *cachedNode) iterateRefs(path []byte, onHashNode func([]byte, common.Hash) error) error {
	return iterateRefs(n.node, path, onHashNode)
}

// iterateRefs traverses the node hierarchy of a cached node and invokes the
// provided callback on all hash nodes.
func iterateRefs(n node, path []byte, onHashNode func([]byte, common.Hash) error) error {
	switch n := n.(type) {
	case *rawShortNode:
		return iterateRefs(n.Val, append(path, n.Key...), onHashNode)
	case rawFullNode:
		for i := 0; i < 16; i++ {
			if err := iterateRefs(n[i], append(path, byte(i)), onHashNode); err != nil {
				return err
			}
		}
		return nil
	case hashNode:
		return onHashNode(path, common.BytesToHash(n))
	case valueNode, nil:
		return nil
	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// cachedNodeSize is the raw size of a cachedNode data structure without any
// node data included. It's an approximate size, but should be a lot better
// than not counting them.
var cachedNodeSize = int(reflect.TypeOf(cachedNode{}).Size())

// cachedNodeChildrenSize is the raw size of an initialized but empty external
// reference map.
const cachedNodeChildrenSize = 48

// rlp returns the raw rlp encoded blob of the cached trie node.
func (n *cachedNode) rlp() []byte {
	blob, err := rlp.EncodeToBytes(n.node)
	if err != nil {
		panic(err)
	}
	return blob
}

// obj returns the decoded and expanded trie node.
func (n *cachedNode) obj(hash common.Hash) node {
	return expandNode(hash[:], n.node)
}

// simplifyNode traverses the hierarchy of an expanded memory node and discards
// all the internal caches, returning a node that only contains the raw data.
func simplifyNode(n node) node {
	switch n := n.(type) {
	case *shortNode:
		// Short nodes discard the flags and cascade
		return &rawShortNode{Key: compactToHex(n.Key), Val: simplifyNode(n.Val)}

	case *fullNode:
		// Full nodes discard the flags and cascade
		node := rawFullNode(n.Children)
		for i := 0; i < len(node); i++ {
			if node[i] != nil {
				node[i] = simplifyNode(node[i])
			}
		}
		return node

	case valueNode, hashNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// expandNode traverses the node hierarchy of a collapsed storage node and converts
// all fields and keys into expanded memory form.
func expandNode(hash hashNode, n node) node {
	switch n := n.(type) {
	case *rawShortNode:
		// Short nodes need key and child expansion
		return &shortNode{
			Key: n.Key,
			Val: expandNode(nil, n.Val),
			flags: nodeFlag{
				hash: hash,
			},
		}

	case rawFullNode:
		// Full nodes need child expansion
		node := &fullNode{
			flags: nodeFlag{
				hash: hash,
			},
		}
		for i := 0; i < len(node.Children); i++ {
			if n[i] != nil {
				node.Children[i] = expandNode(nil, n[i])
			}
		}
		return node

	case valueNode, hashNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// Config defines all necessary options for database.
type Config struct {
	Cache     int    // Memory allowance (MB) to use for caching trie nodes in memory
	Journal   string // Journal of clean cache to survive node restarts
	Preimages bool   // Flag whether the preimage of trie key is recorded
	Pruner    PrunerConfig
}

// NewDatabase creates a new trie database to store ephemeral trie content before
// its written out to disk or garbage collected. No read cache is created, so all
// data retrievals will hit the underlying disk database.
func NewDatabase(diskdb ethdb.KeyValueStore) *Database {
	return NewDatabaseWithConfig(diskdb, nil)
}

// NewDatabaseWithConfig creates a new trie database to store ephemeral trie content
// before its written out to disk or garbage collected. It also acts as a read cache
// for nodes loaded from disk.
func NewDatabaseWithConfig(diskdb ethdb.KeyValueStore, config *Config) *Database {
	var cleans *fastcache.Cache
	if config != nil && config.Cache > 0 {
		if config.Journal == "" {
			cleans = fastcache.New(config.Cache * 1024 * 1024)
		} else {
			cleans = fastcache.LoadFromFileOrNew(config.Journal, config.Cache*1024*1024)
		}
	}
	db := &Database{
		diskdb: diskdb,
		cleans: cleans,
		dirties: map[string]*cachedNode{"": {
			children: make(map[string]uint16),
		}},
	}
	if config == nil || config.Preimages { // TODO(karalabe): Flip to default off in the future
		db.preimages = make(map[common.Hash][]byte)
	}
	if config == nil {
		db.pruner = newPruner(PrunerConfig{Enabled: false}, diskdb)
	} else {
		db.pruner = newPruner(config.Pruner, diskdb)
	}
	return db
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.KeyValueStore {
	return db.diskdb
}

// insert inserts a collapsed trie node into the memory database.
// The blob size must be specified to allow proper size tracking.
// All nodes inserted by this function will be reference tracked
// and in theory should only used for **trie nodes** insertion.
func (db *Database) insert(owner common.Hash, path []byte, hash common.Hash, size int, node node) {
	// If the node's already cached, skip
	key := string(EncodeNodeKey(owner, path, hash))
	if _, ok := db.dirties[key]; ok {
		return
	}
	memcacheDirtyWriteMeter.Mark(int64(size))

	// Create the cached entry for this node
	entry := &cachedNode{
		node:      simplifyNode(node),
		size:      uint16(size),
		flushPrev: db.newest,
	}
	entry.iterateRefs(path, func(childPath []byte, child common.Hash) error {
		byteKey := EncodeNodeKey(owner, childPath, child)
		key := string(byteKey)
		if c := db.dirties[key]; c != nil {
			c.parents++
		}
		return nil
	})
	db.dirties[key] = entry

	// Update the flush-list endpoints
	if db.oldest == metaRoot {
		db.oldest, db.newest = key, key
	} else {
		db.dirties[db.newest].flushNext, db.newest = key, key
	}
	db.dirtiesSize += common.StorageSize(common.HashLength + entry.size)
}

// insertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will NOT make a copy of the slice, only use if the
// preimage will NOT be changed later on.
//
// Note, this method assumes that the database's lock is held!
func (db *Database) insertPreimage(hash common.Hash, preimage []byte) {
	// Short circuit if preimage collection is disabled
	if db.preimages == nil {
		return
	}
	// Track the preimage if a yet unknown one
	if _, ok := db.preimages[hash]; ok {
		return
	}
	db.preimages[hash] = preimage
	db.preimagesSize += common.StorageSize(common.HashLength + len(preimage))
}

// node retrieves a cached trie node from memory, or returns nil if none can be
// found in the memory cache.
func (db *Database) node(owner common.Hash, path []byte, hash common.Hash) node {
	// Retrieve the node from the clean cache if available
	byteKey := EncodeNodeKey(owner, path, hash)
	if db.cleans != nil {
		if enc := db.cleans.Get(nil, byteKey); enc != nil {
			memcacheCleanHitMeter.Mark(1)
			memcacheCleanReadMeter.Mark(int64(len(enc)))
			return mustDecodeNode(hash[:], enc)
		}
	}
	// Retrieve the node from the dirty cache if available
	key := string(byteKey)
	db.lock.RLock()
	dirty := db.dirties[key]
	db.lock.RUnlock()

	if dirty != nil {
		memcacheDirtyHitMeter.Mark(1)
		memcacheDirtyReadMeter.Mark(int64(dirty.size))
		return dirty.obj(hash)
	}
	memcacheDirtyMissMeter.Mark(1)

	// Content unavailable in memory, attempt to retrieve from disk
	enc := rawdb.ReadTrieNode(db.diskdb, byteKey)
	if enc == nil {
		return nil
	}
	if db.cleans != nil {
		db.cleans.Set(byteKey, enc)
		memcacheCleanMissMeter.Mark(1)
		memcacheCleanWriteMeter.Mark(int64(len(enc)))
	}
	return mustDecodeNode(hash[:], enc)
}

// NodeByKey retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (db *Database) NodeByKey(byteKey []byte) ([]byte, error) {
	// Retrieve the node from the clean cache if available
	if db.cleans != nil {
		if enc := db.cleans.Get(nil, byteKey); enc != nil {
			memcacheCleanHitMeter.Mark(1)
			memcacheCleanReadMeter.Mark(int64(len(enc)))
			return enc, nil
		}
	}
	// Retrieve the node from the dirty cache if available
	key := string(byteKey)
	db.lock.RLock()
	dirty := db.dirties[key]
	db.lock.RUnlock()

	if dirty != nil {
		memcacheDirtyHitMeter.Mark(1)
		memcacheDirtyReadMeter.Mark(int64(dirty.size))
		return dirty.rlp(), nil
	}
	memcacheDirtyMissMeter.Mark(1)

	// Content unavailable in memory, attempt to retrieve from disk
	enc := rawdb.ReadTrieNode(db.diskdb, byteKey)
	if len(enc) != 0 {
		if db.cleans != nil {
			db.cleans.Set(byteKey, enc)
			memcacheCleanMissMeter.Mark(1)
			memcacheCleanWriteMeter.Mark(int64(len(enc)))
		}
		return enc, nil
	}
	return nil, errors.New("not found")
}

// Node retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (db *Database) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	// It doesn't make sense to retrieve the metaroot
	if hash == (common.Hash{}) {
		return nil, errors.New("not found")
	}
	return db.NodeByKey(EncodeNodeKey(owner, path, hash))
}

// preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *Database) preimage(hash common.Hash) []byte {
	// Short circuit if preimage collection is disabled
	if db.preimages == nil {
		return nil
	}
	// Retrieve the node from cache if available
	db.lock.RLock()
	preimage := db.preimages[hash]
	db.lock.RUnlock()

	if preimage != nil {
		return preimage
	}
	return rawdb.ReadPreimage(db.diskdb, hash)
}

// Nodes retrieves the hashes of all the nodes cached within the memory database.
// This method is extremely expensive and should only be used to validate internal
// states in test code.
func (db *Database) Nodes() []string {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var keys = make([]string, 0, len(db.dirties))
	for key := range db.dirties {
		if key != metaRoot { // Special case for "root" references/nodes
			keys = append(keys, key)
		}
	}
	return keys
}

// Reference adds a new reference from a parent node to a child node.
// This function is used to add reference between internal trie node
// and external node(e.g. storage trie root), all internal trie nodes
// are referenced together by database itself.
func (db *Database) Reference(owner common.Hash, child common.Hash, parent common.Hash, parentPath []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.reference(owner, child, parent, parentPath)
}

// reference is the private locked version of Reference.
func (db *Database) reference(owner common.Hash, child common.Hash, parent common.Hash, parentPath []byte) {
	// If the node does not exist, it's a node pulled from disk, skip.
	childKey := string(EncodeNodeKey(owner, []byte{}, child))
	node, ok := db.dirties[childKey]
	if !ok {
		return
	}
	// If the reference already exists, only duplicate for roots
	parentKey := string(EncodeNodeKey(common.Hash{}, parentPath, parent))
	if db.dirties[parentKey].children == nil {
		db.dirties[parentKey].children = make(map[string]uint16)
		db.childrenSize += cachedNodeChildrenSize
	} else if _, ok = db.dirties[parentKey].children[childKey]; ok && parent != (common.Hash{}) {
		return
	}
	node.parents++
	db.dirties[parentKey].children[childKey]++
	if db.dirties[parentKey].children[childKey] == 1 {
		db.childrenSize += common.HashLength + 2 // uint16 counter
	}
}

// Dereference removes an existing reference from a root node.
func (db *Database) Dereference(root common.Hash) {
	// Sanity check to ensure that the meta-root is not removed
	if root == (common.Hash{}) {
		log.Error("Attempted to dereference the trie cache meta root")
		return
	}
	db.lock.Lock()
	defer db.lock.Unlock()

	nodes, storage, start := len(db.dirties), db.dirtiesSize, time.Now()
	db.dereference(common.Hash{}, root, []byte{}, common.Hash{}, common.Hash{}, nil)

	db.gcnodes += uint64(nodes - len(db.dirties))
	db.gcsize += storage - db.dirtiesSize
	db.gctime += time.Since(start)

	memcacheGCTimeTimer.Update(time.Since(start))
	memcacheGCSizeMeter.Mark(int64(storage - db.dirtiesSize))
	memcacheGCNodesMeter.Mark(int64(nodes - len(db.dirties)))

	log.Info("Dereferenced trie from memory database", "nodes", nodes-len(db.dirties), "size", storage-db.dirtiesSize, "time", time.Since(start),
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.dirties), "livesize", db.dirtiesSize)
}

// dereference is the private locked version of Dereference.
func (db *Database) dereference(childOwner common.Hash, childHash common.Hash, childPath []byte, parentOwner common.Hash, parentHash common.Hash, parentPath []byte) {
	// Dereference the parent-child
	parentKey := string(EncodeNodeKey(parentOwner, parentPath, parentHash))
	parent := db.dirties[parentKey]

	childKey := string(EncodeNodeKey(childOwner, childPath, childHash))
	if parent.children != nil && parent.children[childKey] > 0 {
		parent.children[childKey]--
		if parent.children[childKey] == 0 {
			delete(parent.children, childKey)
			db.childrenSize -= (common.HashLength + 2) // uint16 counter
		}
	}
	// If the child does not exist, it's a previously committed node.
	child, ok := db.dirties[childKey]
	if !ok {
		return
	}
	// If there are no more references to the child, delete it and cascade
	if child.parents > 0 {
		// This is a special cornercase where a node loaded from disk (i.e. not in the
		// memcache any more) gets reinjected as a new node (short node split into full,
		// then reverted into short), causing a cached node to have no parents. That is
		// no problem in itself, but don't make maxint parents out of it.
		child.parents--
	}
	if child.parents == 0 {
		// Remove the node from the flush-list
		switch childKey {
		case db.oldest:
			db.oldest = child.flushNext
			db.dirties[child.flushNext].flushPrev = metaRoot
		case db.newest:
			db.newest = child.flushPrev
			db.dirties[child.flushPrev].flushNext = metaRoot
		default:
			db.dirties[child.flushPrev].flushNext = child.flushNext
			db.dirties[child.flushNext].flushPrev = child.flushPrev
		}
		// Dereference all children and delete the node
		child.iterateRefs(childPath, func(innerPath []byte, innerHash common.Hash) error {
			db.dereference(childOwner, innerHash, innerPath, childOwner, childHash, childPath)
			return nil
		})
		delete(db.dirties, childKey)
		db.dirtiesSize -= common.StorageSize(common.HashLength + int(child.size))
		if child.children != nil {
			db.childrenSize -= cachedNodeChildrenSize
		}
	}
}

// Cap iteratively flushes old but still referenced trie nodes until the total
// memory usage goes below the given threshold.
//
// Note, this method is a non-synchronized mutator. It is unsafe to call this
// concurrently with other mutators.
func (db *Database) Cap(limit common.StorageSize) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	nodes, storage, start := len(db.dirties), db.dirtiesSize, time.Now()
	batch := db.diskdb.NewBatch()

	// db.dirtiesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted.
	size := db.dirtiesSize + common.StorageSize((len(db.dirties)-1)*cachedNodeSize)
	size += db.childrenSize - common.StorageSize(len(db.dirties[metaRoot].children)*(common.HashLength+2))

	// If the preimage cache got large enough, push to disk. If it's still small
	// leave for later to deduplicate writes.
	flushPreimages := db.preimagesSize > 4*1024*1024
	if flushPreimages {
		if db.preimages == nil {
			log.Error("Attempted to write preimages whilst disabled")
		} else {
			rawdb.WritePreimages(batch, db.preimages)
			if batch.ValueSize() > ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
		}
	}
	// Keep committing nodes from the flush-list until we're below allowance
	oldest := db.oldest
	for size > limit && oldest != metaRoot {
		// Fetch the oldest referenced node and push into the batch
		node := db.dirties[oldest]
		owner, path, hash := DecodeNodeKey([]byte(oldest))
		if err := db.writeNode(batch, owner, path, hash, node.rlp(), true); err != nil {
			return err
		}
		// If we exceeded the ideal batch size, commit and reset
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := db.pruner.flushMarker(batch); err != nil {
				return err
			}
			if err := batch.Write(); err != nil {
				log.Error("Failed to write flush list to disk", "err", err)
				return err
			}
			batch.Reset()
		}
		// Iterate to the next flush item, or abort if the size cap was achieved. Size
		// is the total size, including the useful cached data (hash -> blob), the
		// cache item metadata, as well as external children mappings.
		size -= common.StorageSize(common.HashLength + int(node.size) + cachedNodeSize)
		if node.children != nil {
			size -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashLength+2))
		}
		oldest = node.flushNext
	}
	// Flush out any remainder data from the last batch
	if err := db.pruner.flushMarker(batch); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		log.Error("Failed to write flush list to disk", "err", err)
		return err
	}
	// Write successful, clear out the flushed data
	db.lock.Lock()
	defer db.lock.Unlock()

	if flushPreimages {
		if db.preimages == nil {
			log.Error("Attempted to reset preimage cache whilst disabled")
		} else {
			db.preimages, db.preimagesSize = make(map[common.Hash][]byte), 0
		}
	}
	for db.oldest != oldest {
		node := db.dirties[db.oldest]
		delete(db.dirties, db.oldest)
		db.oldest = node.flushNext

		db.dirtiesSize -= common.StorageSize(common.HashLength + int(node.size))
		if node.children != nil {
			db.childrenSize -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashLength+2))
		}
	}
	if db.oldest != metaRoot {
		db.dirties[db.oldest].flushPrev = metaRoot
	}
	db.flushnodes += uint64(nodes - len(db.dirties))
	db.flushsize += storage - db.dirtiesSize
	db.flushtime += time.Since(start)

	memcacheFlushTimeTimer.Update(time.Since(start))
	memcacheFlushSizeMeter.Mark(int64(storage - db.dirtiesSize))
	memcacheFlushNodesMeter.Mark(int64(nodes - len(db.dirties)))

	log.Info("Persisted nodes from memory database", "nodes", nodes-len(db.dirties), "size", storage-db.dirtiesSize, "time", time.Since(start),
		"flushnodes", db.flushnodes, "flushsize", db.flushsize, "flushtime", db.flushtime, "livenodes", len(db.dirties), "livesize", db.dirtiesSize)

	return nil
}

// Commit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions. As a side
// effect, all pre-images accumulated up to this point are also written.
//
// Note, this method is a non-synchronized mutator. It is unsafe to call this
// concurrently with other mutators.
func (db *Database) Commit(root common.Hash, report bool, callback func(key []byte)) error {
	return db.CommitWithMetadata(0, common.Hash{}, root, report, callback)
}

// CommitWithMetadata is one variation of commit operation with two additional parameters.
// In this function the meta information of the operation will be saved into the disk
// for future pruning processing.
func (db *Database) CommitWithMetadata(number uint64, hash common.Hash, root common.Hash, report bool, callback func(key []byte)) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	start := time.Now()
	batch := db.diskdb.NewBatch()

	// Move all of the accumulated preimages into a write batch
	if db.preimages != nil {
		rawdb.WritePreimages(batch, db.preimages)
		// Since we're going to replay trie node writes into the clean cache, flush out
		// any batched pre-images before continuing.
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
	}
	// Move the trie itself into the batch, flushing if enough data is accumulated
	nodes, storage := len(db.dirties), db.dirtiesSize

	writeMeta := number != 0 && hash != (common.Hash{})
	uncacher := &cleaner{db: db}
	if writeMeta {
		db.pruner.commitStart(number, hash)
	}
	if err := db.commit(common.Hash{}, []byte{}, root, batch, uncacher, callback); err != nil {
		log.Error("Failed to commit trie from trie database", "err", err)
		return err
	}
	// Trie mostly committed to disk, flush any batch leftovers
	if err := db.pruner.flushMarker(batch); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		log.Error("Failed to write trie to disk", "err", err)
		return err
	}
	// Uncache any leftovers in the last batch
	db.lock.Lock()
	defer db.lock.Unlock()

	batch.Replay(uncacher)
	batch.Reset()

	if writeMeta {
		if err := db.pruner.commitEnd(); err != nil {
			log.Warn("Failed to persist commit record", "hash", hash, "number", number, "err", err)
			return err
		}
	}
	// Reset the storage counters, commit bloom and bumped metrics
	if db.preimages != nil {
		db.preimages, db.preimagesSize = make(map[common.Hash][]byte), 0
	}
	memcacheCommitTimeTimer.Update(time.Since(start))
	memcacheCommitSizeMeter.Mark(int64(storage - db.dirtiesSize))
	memcacheCommitNodesMeter.Mark(int64(nodes - len(db.dirties)))

	logger := log.Info
	if !report {
		logger = log.Debug
	}
	logger("Persisted trie from memory database", "nodes", nodes-len(db.dirties)+int(db.flushnodes), "size", storage-db.dirtiesSize+db.flushsize, "time", time.Since(start)+db.flushtime,
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.dirties), "livesize", db.dirtiesSize)

	// Reset the garbage collection statistics
	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0
	db.flushnodes, db.flushsize, db.flushtime = 0, 0, 0

	return nil
}

// commit is the private locked version of Commit.
func (db *Database) commit(owner common.Hash, path []byte, hash common.Hash, batch ethdb.Batch, uncacher *cleaner, callback func(key []byte)) error {
	// If the node does not exist, it's a previously committed node
	byteKey := EncodeNodeKey(owner, path, hash)
	key := string(byteKey)
	node, ok := db.dirties[key]
	if !ok {
		return nil
	}
	if err := node.iterateRefs(path, func(childPath []byte, childHash common.Hash) error {
		return db.commit(owner, childPath, childHash, batch, uncacher, callback)
	}); err != nil {
		return err
	}
	for child := range node.children {
		owner, path, hash := DecodeNodeKey([]byte(child))
		if err := db.commit(owner, path, hash, batch, uncacher, callback); err != nil {
			return err
		}
	}
	if err := db.writeNode(batch, owner, path, hash, node.rlp(), false); err != nil {
		return err
	}
	if callback != nil {
		callback(byteKey)
	}
	// If we've reached an optimal batch size, commit and start over
	if batch.ValueSize() >= ethdb.IdealBatchSize {
		if err := db.pruner.flushMarker(batch); err != nil {
			return err
		}
		if err := batch.Write(); err != nil {
			return err
		}
		db.lock.Lock()
		batch.Replay(uncacher)
		batch.Reset()
		db.lock.Unlock()
	}
	return nil
}

// writeNode wraps the necessary operation of flushing a trie node into the disk.
func (db *Database) writeNode(writer ethdb.KeyValueWriter, owner common.Hash, path []byte, hash common.Hash, node []byte, partial bool) error {
	// Mark no deletion if the written node is in the deletion set of
	// the previous commits.
	key := EncodeNodeKey(owner, path, hash)
	if err := db.pruner.addKey(key, partial); err != nil {
		return err
	}
	// Flush the node index and node itself into the disk in atomic way
	rawdb.WriteTrieNode(writer, key, node)
	return nil
}

// cleaner is a database batch replayer that takes a batch of write operations
// and cleans up the trie database from anything written to disk.
type cleaner struct {
	db *Database
}

// Put reacts to database writes and implements dirty data uncaching. This is the
// post-processing step of a commit operation where the already persisted trie is
// removed from the dirty cache and moved into the clean cache. The reason behind
// the two-phase commit is to ensure ensure data availability while moving from
// memory to disk.
func (c *cleaner) Put(key []byte, rlp []byte) error {
	ok, rawKey := rawdb.IsStateTrieNodeKey(key)
	if !ok {
		return errors.New("unexpected data")
	}
	nodeKey := string(rawKey)

	// If the node does not exist, we're done on this path
	node, ok := c.db.dirties[nodeKey]
	if !ok {
		return nil
	}
	// Node still exists, remove it from the flush-list
	switch nodeKey {
	case c.db.oldest:
		c.db.oldest = node.flushNext
		c.db.dirties[node.flushNext].flushPrev = metaRoot
	case c.db.newest:
		c.db.newest = node.flushPrev
		c.db.dirties[node.flushPrev].flushNext = metaRoot
	default:
		c.db.dirties[node.flushPrev].flushNext = node.flushNext
		c.db.dirties[node.flushNext].flushPrev = node.flushPrev
	}
	// Remove the node from the dirty cache
	delete(c.db.dirties, nodeKey)
	c.db.dirtiesSize -= common.StorageSize(common.HashLength + int(node.size))
	if node.children != nil {
		c.db.dirtiesSize -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashLength+2))
	}
	// Move the flushed node into the clean cache to prevent insta-reloads
	if c.db.cleans != nil {
		c.db.cleans.Set(rawKey, rlp)
		memcacheCleanWriteMeter.Mark(int64(len(rlp)))
	}
	return nil
}

func (c *cleaner) Delete(key []byte) error {
	panic("not implemented")
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *Database) Size() (common.StorageSize, common.StorageSize) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// db.dirtiesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted.
	var metadataSize = common.StorageSize((len(db.dirties) - 1) * cachedNodeSize)
	var metarootRefs = common.StorageSize(len(db.dirties[metaRoot].children) * (common.HashLength + 2))
	return db.dirtiesSize + db.childrenSize + metadataSize - metarootRefs, db.preimagesSize
}

// saveCache saves clean state cache to given directory path
// using specified CPU cores.
func (db *Database) saveCache(dir string, threads int) error {
	if db.cleans == nil {
		return nil
	}
	log.Info("Writing clean trie cache to disk", "path", dir, "threads", threads)

	start := time.Now()
	err := db.cleans.SaveToFileConcurrent(dir, threads)
	if err != nil {
		log.Error("Failed to persist clean trie cache", "error", err)
		return err
	}
	log.Info("Persisted the clean trie cache", "path", dir, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// SaveCache atomically saves fast cache data to the given dir using all
// available CPU cores.
func (db *Database) SaveCache(dir string) error {
	return db.saveCache(dir, runtime.GOMAXPROCS(0))
}

// SaveCachePeriodically atomically saves fast cache data to the given dir with
// the specified interval. All dump operation will only use a single CPU core.
func (db *Database) SaveCachePeriodically(dir string, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.saveCache(dir, 1)
		case <-stopCh:
			return
		}
	}
}

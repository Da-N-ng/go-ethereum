// Copyright 2016 The go-ethereum Authors
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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"crypto/ecdsa"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
)

type LesServer struct {
	config          *eth.Config
	protocolManager *ProtocolManager
	fcManager       *flowcontrol.ClientManager // nil if our node is client only
	fcCostStats     *requestCostStats
	defParams       *flowcontrol.ServerParams
	lesTopics       []discv5.Topic
	privateKey      *ecdsa.PrivateKey
	quitSync        chan struct{}

	// Checkpoint relative fields
	registrar        *registrar.Registrar     // Handler for checkpoint contract
	stableCheckpoint *light.TrustedCheckpoint // The nearest stable checkpoint

	// Indexers
	chtIndexer       *core.ChainIndexer // Indexers for creating cht root for each block section
	bloomTrieIndexer *core.ChainIndexer // Indexers for creating bloom trie root for each block section
}

func NewLesServer(eth *eth.Ethereum, config *eth.Config) (*LesServer, error) {
	quitSync := make(chan struct{})
	pm, err := NewProtocolManager(eth.BlockChain().Config(), light.DefaultServerIndexerConfig, false, ServerProtocolVersions, config.NetworkId, eth.EventMux(), eth.Engine(), newPeerSet(), eth.BlockChain(), eth.TxPool(), eth.ChainDb(), nil, nil, nil, quitSync, new(sync.WaitGroup))
	if err != nil {
		return nil, err
	}

	lesTopics := make([]discv5.Topic, len(AdvertiseProtocolVersions))
	for i, pv := range AdvertiseProtocolVersions {
		lesTopics[i] = lesTopic(eth.BlockChain().Genesis().Hash(), pv)
	}

	srv := &LesServer{
		config:          config,
		protocolManager: pm,
		quitSync:        quitSync,
		lesTopics:       lesTopics,
		chtIndexer:      light.NewChtIndexer(eth.ChainDb(), params.CHTFrequencyServer, params.HelperTrieProcessConfirmations),
		bloomTrieIndexer: light.NewBloomTrieIndexer(eth.ChainDb(), params.BloomBitsBlocks, params.BloomConfirms,
			params.BloomTrieFrequency, params.HelperTrieProcessConfirmations),
	}
	logger := log.New()

	chtV1SectionCount, _, _ := srv.chtIndexer.Sections() // indexer still uses LES/1 4k section size for backwards server compatibility
	chtV2SectionCount := chtV1SectionCount / (params.CHTFrequencyClient / params.CHTFrequencyServer)
	if chtV2SectionCount != 0 {
		// convert to LES/2 section
		chtLastSection := chtV2SectionCount - 1
		// convert last LES/2 section index back to LES/1 index for chtIndexer.SectionHead
		chtLastSectionV1 := (chtLastSection+1)*(params.CHTFrequencyClient/params.CHTFrequencyServer) - 1
		chtSectionHead := srv.chtIndexer.SectionHead(chtLastSectionV1)
		chtRoot := light.GetChtRoot(pm.chainDb, chtLastSectionV1, chtSectionHead)
		logger.Info("Loaded CHT", "section", chtLastSection, "head", chtSectionHead, "root", chtRoot)
	}
	bloomTrieSectionCount, _, _ := srv.bloomTrieIndexer.Sections()
	if bloomTrieSectionCount != 0 {
		bloomTrieLastSection := bloomTrieSectionCount - 1
		bloomTrieSectionHead := srv.bloomTrieIndexer.SectionHead(bloomTrieLastSection)
		bloomTrieRoot := light.GetBloomTrieRoot(pm.chainDb, bloomTrieLastSection, bloomTrieSectionHead)
		logger.Info("Loaded bloom trie", "section", bloomTrieLastSection, "head", bloomTrieSectionHead, "root", bloomTrieRoot)
	}

	srv.chtIndexer.Start(eth.BlockChain())
	pm.server = srv

	srv.defParams = &flowcontrol.ServerParams{
		BufLimit:    300000000,
		MinRecharge: 50000,
	}
	srv.fcManager = flowcontrol.NewClientManager(uint64(config.LightServ), 10, 1000000000)
	srv.fcCostStats = newCostStats(eth.ChainDb())
	if addr, ok := registrar.RegistrarAddr[eth.BlockChain().Genesis().Hash()]; ok {
		registrar, err := registrar.NewRegistrar(addr, eth.APIBackend, false)
		if err != nil {
			return nil, err
		}
		srv.registrar = registrar
	}
	return srv, nil
}

func (s *LesServer) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start starts the LES server
func (s *LesServer) Start(srvr *p2p.Server) {
	s.protocolManager.Start(s.config.LightPeers)
	if srvr.DiscV5 != nil {
		for _, topic := range s.lesTopics {
			topic := topic
			go func() {
				logger := log.New("topic", topic)
				logger.Info("Starting topic registration")
				defer logger.Info("Terminated topic registration")

				srvr.DiscV5.RegisterTopic(topic, s.quitSync)
			}()
		}
	}
	s.privateKey = srvr.PrivateKey
	s.protocolManager.blockLoop()
	if s.registrar != nil {
		s.checkpointLoop()
	}
}

func (s *LesServer) SetBloomBitsIndexer(bloomIndexer *core.ChainIndexer) {
	bloomIndexer.AddChildIndexer(s.bloomTrieIndexer)
}

// Stop stops the LES service
func (s *LesServer) Stop() {
	s.chtIndexer.Close()
	// bloom trie indexer is closed by parent bloombits indexer
	s.fcCostStats.store()
	s.fcManager.Stop()
	go func() {
		<-s.protocolManager.noMorePeers
	}()
	s.protocolManager.Stop()
	close(s.quitSync)
}

// checkpointLoop starts a standalone goroutine to watch new checkpoint event and upgrades local's stable checkpoint.
func (s *LesServer) checkpointLoop() (err error) {
	sink := make(chan *contract.ContractNewCheckpointEvent)
	sub, err := s.registrar.WatchNewCheckpointEvent(sink)
	if err != nil {
		return
	}
	defer func() {
		sub.Unsubscribe()
	}()

	for {
		select {
		case event := <-sink:
			// New stable checkpoint received
			// Note several duplicate events can be received due to chain reorg, just track the first arrive one.
			if event.Index.Uint64() > s.stableCheckpoint.SectionIdx {
				checkpoint := &light.TrustedCheckpoint{
					SectionIdx:    event.Index.Uint64(),
					SectionHead:   common.Hash(event.SectionHead),
					ChtRoot:       common.Hash(event.ChtRoot),
					BloomTrieRoot: common.Hash(event.BloomTrieRoot),
				}
				light.WriteTrustedCheckpoint(s.protocolManager.chainDb, checkpoint)
				s.stableCheckpoint = checkpoint
				log.Info("update checkpoint", "section", checkpoint.SectionIdx, "head", checkpoint.SectionHead.Hex(),
					"chtRoot", checkpoint.ChtRoot.Hex(), "bloomTrieRoot", checkpoint.BloomTrieRoot.Hex())
			}

		case <-s.quitSync:
			// Les server is closed.
			return
		}
	}
}

func (pm *ProtocolManager) blockLoop() {
	pm.wg.Add(1)
	headCh := make(chan core.ChainHeadEvent, 10)
	headSub := pm.blockchain.SubscribeChainHeadEvent(headCh)
	go func() {
		var lastHead *types.Header
		lastBroadcastTd := common.Big0
		for {
			select {
			case ev := <-headCh:
				peers := pm.peers.AllPeers()
				if len(peers) > 0 {
					header := ev.Block.Header()
					hash := header.Hash()
					number := header.Number.Uint64()
					td := rawdb.ReadTd(pm.chainDb, hash, number)
					if td != nil && td.Cmp(lastBroadcastTd) > 0 {
						var reorg uint64
						if lastHead != nil {
							reorg = lastHead.Number.Uint64() - rawdb.FindCommonAncestor(pm.chainDb, header, lastHead).Number.Uint64()
						}
						lastHead = header
						lastBroadcastTd = td

						log.Debug("Announcing block to peers", "number", number, "hash", hash, "td", td, "reorg", reorg)

						announce := announceData{Hash: hash, Number: number, Td: td, ReorgDepth: reorg}
						var (
							signed         bool
							signedAnnounce announceData
						)

						for _, p := range peers {
							switch p.announceType {

							case announceTypeSimple:
								select {
								case p.announceChn <- announce:
								default:
									pm.removePeer(p.id)
								}

							case announceTypeSigned:
								if !signed {
									signedAnnounce = announce
									signedAnnounce.sign(pm.server.privateKey)
									signed = true
								}

								select {
								case p.announceChn <- signedAnnounce:
								default:
									pm.removePeer(p.id)
								}
							}
						}
					}
				}
			case <-pm.quitSync:
				headSub.Unsubscribe()
				pm.wg.Done()
				return
			}
		}
	}()
}

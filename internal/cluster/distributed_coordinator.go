package cluster

import (
	"context"
	"log"
	"sync"
	"time"

	"task-scheduler/internal/types"
)

type DistributedCoordinator struct {
	nodeID    string
	peers     map[string]*Peer
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	leader    bool
	leaderID  string
	heartbeat time.Duration
	timeout   time.Duration
}

type Peer struct {
	ID       string
	Address  string
	LastSeen time.Time
	Status   string
	Load     int
	IsLeader bool
}

func NewDistributedCoordinator(nodeID string, heartbeat, timeout time.Duration) *DistributedCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &DistributedCoordinator{
		nodeID:    nodeID,
		peers:     make(map[string]*Peer),
		ctx:       ctx,
		cancel:    cancel,
		heartbeat: heartbeat,
		timeout:   timeout,
	}
}

func (dc *DistributedCoordinator) Start() {
	go dc.heartbeatLoop()
	go dc.peerDiscovery()
	go dc.loadBalancing()
}

func (dc *DistributedCoordinator) Stop() {
	dc.cancel()
}

func (dc *DistributedCoordinator) heartbeatLoop() {
	ticker := time.NewTicker(dc.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.broadcastHeartbeat()
		}
	}
}

func (dc *DistributedCoordinator) broadcastHeartbeat() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	for _, peer := range dc.peers {
		if now.Sub(peer.LastSeen) > dc.timeout {
			peer.Status = "offline"
		}
	}
}

func (dc *DistributedCoordinator) peerDiscovery() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.discoverPeers()
		}
	}
}

func (dc *DistributedCoordinator) discoverPeers() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	for _, peer := range dc.peers {
		if peer.Status == "offline" {
			log.Printf("Peer %s is offline", peer.ID)
		}
	}
}

func (dc *DistributedCoordinator) loadBalancing() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.rebalanceLoad()
		}
	}
}

func (dc *DistributedCoordinator) rebalanceLoad() {
	dc.mu.RLock()
	peers := make([]*Peer, 0, len(dc.peers))
	for _, peer := range dc.peers {
		if peer.Status == "online" {
			peers = append(peers, peer)
		}
	}
	dc.mu.RUnlock()

	if len(peers) == 0 {
		return
	}

	avgLoad := dc.calculateAverageLoad(peers)
	for _, peer := range peers {
		if peer.Load > avgLoad*2 {
			log.Printf("High load detected on peer %s: %d", peer.ID, peer.Load)
		}
	}
}

func (dc *DistributedCoordinator) calculateAverageLoad(peers []*Peer) int {
	if len(peers) == 0 {
		return 0
	}

	total := 0
	for _, peer := range peers {
		total += peer.Load
	}
	return total / len(peers)
}

func (dc *DistributedCoordinator) AddPeer(id, address string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.peers[id] = &Peer{
		ID:       id,
		Address:  address,
		LastSeen: time.Now(),
		Status:   "online",
		Load:     0,
	}
}

func (dc *DistributedCoordinator) RemovePeer(id string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	delete(dc.peers, id)
}

func (dc *DistributedCoordinator) UpdatePeerLoad(id string, load int) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if peer, exists := dc.peers[id]; exists {
		peer.Load = load
		peer.LastSeen = time.Now()
	}
}

func (dc *DistributedCoordinator) GetLeastLoadedPeer() string {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	var leastLoaded *Peer
	for _, peer := range dc.peers {
		if peer.Status == "online" && (leastLoaded == nil || peer.Load < leastLoaded.Load) {
			leastLoaded = peer
		}
	}

	if leastLoaded != nil {
		return leastLoaded.ID
	}
	return ""
}

func (dc *DistributedCoordinator) SetLeader(leaderID string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.leaderID = leaderID
	dc.leader = (leaderID == dc.nodeID)

	for _, peer := range dc.peers {
		peer.IsLeader = (peer.ID == leaderID)
	}
}

func (dc *DistributedCoordinator) IsLeader() bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.leader
}

func (dc *DistributedCoordinator) GetLeaderID() string {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.leaderID
}

func (dc *DistributedCoordinator) GetPeers() []*types.NodeInfo {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	peers := make([]*types.NodeInfo, 0, len(dc.peers))
	for _, peer := range dc.peers {
		peers = append(peers, &types.NodeInfo{
			ID:       peer.ID,
			Address:  peer.Address,
			IsLeader: peer.IsLeader,
			LastSeen: peer.LastSeen,
			Status:   peer.Status,
		})
	}
	return peers
}

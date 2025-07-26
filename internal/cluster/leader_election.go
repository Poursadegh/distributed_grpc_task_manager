package cluster

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type LeaderElection struct {
	raft           *raft.Raft
	nodeID         string
	address        string
	dataDir        string
	peers          []string
	onLeaderChange func(bool)
}

func NewLeaderElection(nodeID, address, dataDir string, peers []string, onLeaderChange func(bool)) *LeaderElection {
	return &LeaderElection{
		nodeID:         nodeID,
		address:        address,
		dataDir:        dataDir,
		peers:          peers,
		onLeaderChange: onLeaderChange,
	}
}

func (le *LeaderElection) Start(ctx context.Context) error {
	// Create data directory
	if err := os.MkdirAll(le.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(le.nodeID)
	config.SnapshotInterval = 30 * time.Second
	config.SnapshotThreshold = 1000
	config.HeartbeatTimeout = 1000 * time.Millisecond
	config.ElectionTimeout = 1000 * time.Millisecond
	config.CommitTimeout = 5 * time.Second

	// Create transport
	addr, err := net.ResolveTCPAddr("tcp", le.address)
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	transport, err := raft.NewTCPTransport(le.address, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	// Create log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(le.dataDir, "raft.db"))
	if err != nil {
		return fmt.Errorf("failed to create log store: %w", err)
	}

	// Create stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(le.dataDir, "stable.db"))
	if err != nil {
		return fmt.Errorf("failed to create stable store: %w", err)
	}

	// Create snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(le.dataDir, 3, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Create FSM (Finite State Machine)
	fsm := &ClusterFSM{}

	// Create Raft instance
	le.raft, err = raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("failed to create raft: %w", err)
	}

	// Bootstrap if this is the first node
	if len(le.peers) == 0 {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		le.raft.BootstrapCluster(configuration)
	} else {
		// Join existing cluster
		for _, peer := range le.peers {
			if err := le.raft.AddVoter(raft.ServerID(peer), raft.ServerAddress(peer), 0, 0).Error(); err != nil {
				// Ignore errors if already part of cluster
				fmt.Printf("Warning: failed to add voter %s: %v\n", peer, err)
			}
		}
	}

	// Start monitoring leader changes
	go le.monitorLeaderChanges(ctx)

	return nil
}

// Stop stops the Raft node
func (le *LeaderElection) Stop() error {
	if le.raft != nil {
		return le.raft.Shutdown().Error()
	}
	return nil
}

// IsLeader checks if this node is the leader
func (le *LeaderElection) IsLeader() bool {
	if le.raft == nil {
		return false
	}
	return le.raft.State() == raft.Leader
}

// GetLeader returns the current leader ID
func (le *LeaderElection) GetLeader() string {
	if le.raft == nil {
		return ""
	}
	return string(le.raft.Leader())
}

// GetState returns the current state of this node
func (le *LeaderElection) GetState() string {
	if le.raft == nil {
		return "stopped"
	}
	return le.raft.State().String()
}

// GetPeers returns the list of peers
func (le *LeaderElection) GetPeers() []raft.Server {
	if le.raft == nil {
		return nil
	}
	return le.raft.GetConfiguration().Configuration().Servers
}

// monitorLeaderChanges monitors for leader changes and calls the callback
func (le *LeaderElection) monitorLeaderChanges(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var wasLeader bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			isLeader := le.IsLeader()
			if isLeader != wasLeader {
				wasLeader = isLeader
				if le.onLeaderChange != nil {
					le.onLeaderChange(isLeader)
				}
			}
		}
	}
}

// ClusterFSM implements the Raft FSM interface
type ClusterFSM struct {
	mu    sync.RWMutex
	state map[string]interface{}
}

// Apply applies a log entry to the FSM
func (c *ClusterFSM) Apply(log *raft.Log) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For now, we'll just store the command
	// In a real implementation, you'd parse and apply specific commands
	c.state[string(log.Index)] = log.Data

	return nil
}

// Snapshot creates a snapshot of the FSM
func (c *ClusterFSM) Snapshot() (raft.FSMSnapshot, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy of the state
	stateCopy := make(map[string]interface{})
	for k, v := range c.state {
		stateCopy[k] = v
	}

	return &ClusterSnapshot{state: stateCopy}, nil
}

// Restore restores the FSM from a snapshot
func (c *ClusterFSM) Restore(rc io.ReadCloser) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear current state
	c.state = make(map[string]interface{})

	// Read and restore state
	decoder := gob.NewDecoder(rc)
	return decoder.Decode(&c.state)
}

// ClusterSnapshot implements the FSM snapshot interface
type ClusterSnapshot struct {
	state map[string]interface{}
}

// Persist persists the snapshot
func (c *ClusterSnapshot) Persist(sink raft.SnapshotSink) error {
	encoder := gob.NewEncoder(sink)
	if err := encoder.Encode(c.state); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// Release releases the snapshot
func (c *ClusterSnapshot) Release() {
	// Nothing to do for this simple implementation
}

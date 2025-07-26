package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type LeaderElection struct {
	raft           *raft.Raft
	nodeID         string
	address        string
	peers          []string
	onLeaderChange func(bool)
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewLeaderElection(nodeID, address string, peers []string, onLeaderChange func(bool)) *LeaderElection {
	ctx, cancel := context.WithCancel(context.Background())
	return &LeaderElection{
		nodeID:         nodeID,
		address:        address,
		peers:          peers,
		onLeaderChange: onLeaderChange,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (le *LeaderElection) Start() error {
	dataDir := filepath.Join("data", "raft", le.nodeID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(le.nodeID)

	addr, err := net.ResolveTCPAddr("tcp", le.address)
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	transport, err := raft.NewTCPTransport(le.address, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
	if err != nil {
		return fmt.Errorf("failed to create log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "stable.db"))
	if err != nil {
		return fmt.Errorf("failed to create stable store: %w", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(dataDir, 3, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %w", err)
	}

	fsm := &ClusterFSM{}

	le.raft, err = raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("failed to create raft: %w", err)
	}

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
		for _, peer := range le.peers {
			err := le.raft.AddVoter(raft.ServerID(peer), raft.ServerAddress(peer), 0, 0).Error()
			if err != nil && err.Error() != "server already exists" {
				log.Printf("Failed to add peer %s: %v", peer, err)
			}
		}
	}

	go le.monitorLeaderChanges()

	log.Printf("Raft node %s started on %s", le.nodeID, le.address)
	return nil
}

func (le *LeaderElection) Stop() {
	if le.raft != nil {
		le.raft.Shutdown()
	}
	le.cancel()
}

func (le *LeaderElection) IsLeader() bool {
	return le.raft != nil && le.raft.State() == raft.Leader
}

func (le *LeaderElection) GetLeader() string {
	if le.raft == nil {
		return ""
	}
	return string(le.raft.Leader())
}

func (le *LeaderElection) GetState() string {
	if le.raft == nil {
		return "unknown"
	}
	return le.raft.State().String()
}

func (le *LeaderElection) GetPeers() []string {
	if le.raft == nil {
		return nil
	}

	config := le.raft.GetConfiguration()
	if err := config.Error(); err != nil {
		return nil
	}

	var peers []string
	for _, server := range config.Configuration().Servers {
		peers = append(peers, string(server.ID))
	}
	return peers
}

func (le *LeaderElection) monitorLeaderChanges() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var wasLeader bool
	for {
		select {
		case <-le.ctx.Done():
			return
		case <-ticker.C:
			isLeader := le.IsLeader()
			if isLeader != wasLeader {
				log.Printf("Node %s leadership changed: %t", le.nodeID, isLeader)
				if le.onLeaderChange != nil {
					le.onLeaderChange(isLeader)
				}
				wasLeader = isLeader
			}
		}
	}
}

type ClusterFSM struct {
	state map[string]interface{}
}

func (c *ClusterFSM) Apply(log *raft.Log) interface{} {
	c.state[fmt.Sprintf("%d", log.Index)] = string(log.Data)
	return nil
}

func (c *ClusterFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &ClusterSnapshot{state: c.state}, nil
}

func (c *ClusterFSM) Restore(rc io.ReadCloser) error {
	c.state = make(map[string]interface{})

	decoder := json.NewDecoder(rc)
	return decoder.Decode(&c.state)
}

type ClusterSnapshot struct {
	state map[string]interface{}
}

func (c *ClusterSnapshot) Persist(sink raft.SnapshotSink) error {
	encoder := json.NewEncoder(sink)
	if err := encoder.Encode(c.state); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (c *ClusterSnapshot) Release() {
}

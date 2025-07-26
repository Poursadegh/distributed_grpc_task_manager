package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"task-scheduler/internal/cluster"
	"task-scheduler/internal/queue"
	"task-scheduler/internal/storage"
	"task-scheduler/internal/types"
	"task-scheduler/internal/worker"
)

type Scheduler struct {
	nodeID         string
	address        string
	queue          *queue.PriorityQueue
	storage        storage.Storage
	leaderElection *cluster.LeaderElection
	coordinator    *cluster.DistributedCoordinator
	workerPool     *worker.WorkerPool
	processor      *TaskProcessor
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	isLeader       bool
	peers          []string
}

type TaskProcessor struct {
	storage storage.Storage
}

func NewTaskProcessor(storage storage.Storage) *TaskProcessor {
	return &TaskProcessor{
		storage: storage,
	}
}

func (tp *TaskProcessor) ProcessTask(ctx context.Context, task *types.Task) error {
	log.Printf("Processing task %s with payload: %s", task.ID, string(task.Payload))

	var sleepTime time.Duration
	switch task.Priority {
	case types.PriorityHigh:
		sleepTime = 1 * time.Second
	case types.PriorityMedium:
		sleepTime = 2 * time.Second
	case types.PriorityLow:
		sleepTime = 3 * time.Second
	}

	time.Sleep(sleepTime)

	log.Printf("Task %s completed successfully", task.ID)

	return nil
}

func NewScheduler(nodeID, address string, storage storage.Storage, peers []string) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	queue := queue.NewPriorityQueue()
	processor := NewTaskProcessor(storage)
	coordinator := cluster.NewDistributedCoordinator(nodeID, 5*time.Second, 30*time.Second)

	return &Scheduler{
		nodeID:      nodeID,
		address:     address,
		queue:       queue,
		storage:     storage,
		processor:   processor,
		coordinator: coordinator,
		ctx:         ctx,
		cancel:      cancel,
		peers:       peers,
	}
}

func (s *Scheduler) Start() error {
	log.Printf("Starting scheduler node %s on %s", s.nodeID, s.address)

	dataDir := fmt.Sprintf("./data/%s", s.nodeID)
	s.leaderElection = cluster.NewLeaderElection(s.nodeID, s.address, dataDir, s.peers, s.onLeaderChange)

	if err := s.leaderElection.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start leader election: %w", err)
	}

	s.workerPool = worker.NewWorkerPool(s.processor, s.storage, s.queue, 5)

	if err := s.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	s.coordinator.Start()

	go s.recoveryProcess()
	go s.heartbeat()
	go s.distributedTaskDistribution()

	log.Printf("Scheduler started successfully")
	return nil
}

func (s *Scheduler) Stop() {
	log.Printf("Stopping scheduler node %s", s.nodeID)

	s.cancel()

	if s.workerPool != nil {
		s.workerPool.Stop()
	}

	if s.leaderElection != nil {
		s.leaderElection.Stop()
	}

	if s.coordinator != nil {
		s.coordinator.Stop()
	}

	log.Printf("Scheduler stopped")
}

func (s *Scheduler) SubmitTask(priority types.Priority, payload json.RawMessage) (*types.Task, error) {
	if !s.isLeader {
		return nil, fmt.Errorf("only leader can submit tasks")
	}

	task := types.NewTask(priority, payload)

	ctx := context.Background()
	if err := s.storage.SaveTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	s.queue.Add(task)

	if err := s.storage.SaveToQueue(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task to queue: %w", err)
	}

	log.Printf("Submitted task %s with priority %s", task.ID, priority.String())
	return task, nil
}

func (s *Scheduler) GetTask(id string) (*types.Task, error) {
	ctx := context.Background()
	return s.storage.GetTask(ctx, id)
}

func (s *Scheduler) GetTasksByStatus(status types.Status) ([]*types.Task, error) {
	ctx := context.Background()
	return s.storage.GetTasksByStatus(ctx, status)
}

func (s *Scheduler) GetAllTasks() ([]*types.Task, error) {
	ctx := context.Background()
	return s.storage.GetAllTasks(ctx)
}

func (s *Scheduler) GetQueueStats() map[string]interface{} {
	return s.queue.GetStats()
}

func (s *Scheduler) GetWorkerMetrics() *worker.WorkerMetrics {
	return s.workerPool.GetMetrics()
}

func (s *Scheduler) IsLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLeader
}

func (s *Scheduler) GetNodeID() string {
	return s.nodeID
}

func (s *Scheduler) GetClusterInfo() *types.ClusterInfo {
	ctx := context.Background()

	nodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("Failed to get nodes: %v", err)
		nodes = []*types.NodeInfo{}
	}

	allTasks, err := s.GetAllTasks()
	if err != nil {
		log.Printf("Failed to get all tasks: %v", err)
		allTasks = []*types.Task{}
	}

	pendingTasks, err := s.GetTasksByStatus(types.StatusPending)
	if err != nil {
		log.Printf("Failed to get pending tasks: %v", err)
		pendingTasks = []*types.Task{}
	}

	runningTasks, err := s.GetTasksByStatus(types.StatusRunning)
	if err != nil {
		log.Printf("Failed to get running tasks: %v", err)
		runningTasks = []*types.Task{}
	}

	return &types.ClusterInfo{
		Nodes:        nodes,
		LeaderID:     s.leaderElection.GetLeader(),
		TotalTasks:   len(allTasks),
		PendingTasks: len(pendingTasks),
		RunningTasks: len(runningTasks),
	}
}

func (s *Scheduler) onLeaderChange(isLeader bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isLeader = isLeader
	log.Printf("Node %s leadership changed: isLeader=%v", s.nodeID, isLeader)

	if isLeader {
		go s.leaderResponsibilities()
	}
}

func (s *Scheduler) leaderResponsibilities() {
	log.Printf("Node %s is now leader, starting leader responsibilities", s.nodeID)

	s.updateNodeInfo()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateNodeInfo()
		}
	}
}

func (s *Scheduler) updateNodeInfo() {
	nodeInfo := &types.NodeInfo{
		ID:       s.nodeID,
		Address:  s.address,
		IsLeader: s.isLeader,
		LastSeen: time.Now(),
		Status:   "active",
	}

	ctx := context.Background()
	if err := s.storage.SaveNodeInfo(ctx, nodeInfo); err != nil {
		log.Printf("Failed to save node info: %v", err)
	}
}

func (s *Scheduler) heartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateNodeInfo()
		}
	}
}

func (s *Scheduler) recoveryProcess() {
	log.Printf("Starting recovery process")

	ctx := context.Background()

	pendingTasks, err := s.storage.GetQueueTasks(ctx)
	if err != nil {
		log.Printf("Failed to get pending tasks during recovery: %v", err)
		return
	}

	for _, task := range pendingTasks {
		if task.Status == types.StatusPending {
			s.queue.Add(task)
			log.Printf("Recovered task %s", task.ID)
		}
	}

	log.Printf("Recovery process completed, recovered %d tasks", len(pendingTasks))
}

func (s *Scheduler) distributedTaskDistribution() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.distributeTasks()
		}
	}
}

func (s *Scheduler) distributeTasks() {
	if !s.isLeader {
		return
	}

	queueSize := s.queue.Len()
	if queueSize == 0 {
		return
	}

	leastLoadedPeer := s.coordinator.GetLeastLoadedPeer()
	if leastLoadedPeer != "" && leastLoadedPeer != s.nodeID {
		log.Printf("Distributing tasks to peer %s", leastLoadedPeer)
	}

	s.coordinator.UpdatePeerLoad(s.nodeID, queueSize)
}

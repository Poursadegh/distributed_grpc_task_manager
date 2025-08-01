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

	maxQueueSize    int
	failureDetector *FailureDetector
	taskReassigner  *TaskReassigner
	autoscaler      *Autoscaler
}

type TaskProcessor struct {
	storage storage.Storage
}

type FailureDetector struct {
	mu              sync.RWMutex
	failedWorkers   map[string]time.Time
	failedNodes     map[string]time.Time
	checkInterval   time.Duration
	cleanupInterval time.Duration
}

type TaskReassigner struct {
	mu               sync.RWMutex
	failedTasks      map[string]*types.Task
	reassignInterval time.Duration
}

type Autoscaler struct {
	mu                 sync.RWMutex
	minWorkers         int
	maxWorkers         int
	scaleUpThreshold   float64
	scaleDownThreshold float64
	checkInterval      time.Duration
	lastScaleTime      time.Time
	cooldownPeriod     time.Duration
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
		nodeID:       nodeID,
		address:      address,
		queue:        queue,
		storage:      storage,
		processor:    processor,
		coordinator:  coordinator,
		ctx:          ctx,
		cancel:       cancel,
		peers:        peers,
		maxQueueSize: 10000,
		failureDetector: &FailureDetector{
			failedWorkers:   make(map[string]time.Time),
			failedNodes:     make(map[string]time.Time),
			checkInterval:   10 * time.Second,
			cleanupInterval: 60 * time.Second,
		},
		taskReassigner: &TaskReassigner{
			failedTasks:      make(map[string]*types.Task),
			reassignInterval: 30 * time.Second,
		},
		autoscaler: &Autoscaler{
			mu:                 sync.RWMutex{},
			minWorkers:         2,
			maxWorkers:         20,
			scaleUpThreshold:   0.8,
			scaleDownThreshold: 0.2,
			checkInterval:      30 * time.Second,
			cooldownPeriod:     60 * time.Second,
		},
	}
}

func (s *Scheduler) Start() error {
	log.Printf("Starting scheduler node %s on %s", s.nodeID, s.address)

	s.leaderElection = cluster.NewLeaderElection(s.nodeID, s.address, s.peers, s.onLeaderChange)

	if err := s.leaderElection.Start(); err != nil {
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
	go s.failureDetection()
	go s.taskReassignment()
	go s.backpressureMonitoring()
	go s.autoscaling()

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
	if s.queue.Len() >= s.maxQueueSize {
		return nil, fmt.Errorf("queue is full, cannot accept new tasks")
	}

	if !s.isLeader {
		return nil, fmt.Errorf("only leader can submit tasks")
	}

	task := types.NewTask(priority, payload)
	return s.submitTaskInternal(task)
}

func (s *Scheduler) SubmitTaskWithDependencies(priority types.Priority, payload json.RawMessage, dependencies []string) (*types.Task, error) {
	if s.queue.Len() >= s.maxQueueSize {
		return nil, fmt.Errorf("queue is full, cannot accept new tasks")
	}

	if !s.isLeader {
		return nil, fmt.Errorf("only leader can submit tasks")
	}

	task := types.NewTaskWithDependencies(priority, payload, dependencies)
	return s.submitTaskInternal(task)
}

func (s *Scheduler) SubmitTaskWithTimeout(priority types.Priority, payload json.RawMessage, maxDuration time.Duration) (*types.Task, error) {
	if s.queue.Len() >= s.maxQueueSize {
		return nil, fmt.Errorf("queue is full, cannot accept new tasks")
	}

	if !s.isLeader {
		return nil, fmt.Errorf("only leader can submit tasks")
	}

	task := types.NewTaskWithTimeout(priority, payload, maxDuration)
	return s.submitTaskInternal(task)
}

func (s *Scheduler) submitTaskInternal(task *types.Task) (*types.Task, error) {
	ctx := context.Background()
	if err := s.storage.SaveTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	s.queue.Add(task)

	if err := s.storage.SaveToQueue(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task to queue: %w", err)
	}

	log.Printf("Submitted task %s with priority %s", task.ID, task.Priority.String())
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

func (s *Scheduler) UpdateTask(task *types.Task) error {
	ctx := context.Background()
	return s.storage.UpdateTask(ctx, task)
}

func (s *Scheduler) DeleteTask(id string) error {
	ctx := context.Background()
	return s.storage.DeleteTask(ctx, id)
}

func (s *Scheduler) GetQueueStats() map[string]interface{} {
	stats := s.queue.GetStats()
	stats["max_queue_size"] = s.maxQueueSize
	stats["queue_utilization"] = float64(stats["total_tasks"].(int)) / float64(s.maxQueueSize)
	return stats
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

func (s *Scheduler) failureDetection() {
	ticker := time.NewTicker(s.failureDetector.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkFailures()
		}
	}
}

func (s *Scheduler) checkFailures() {
	metrics := s.workerPool.GetMetrics()
	if metrics.TasksFailed > 0 {
		log.Printf("Detected %d failed tasks", metrics.TasksFailed)
	}

	ctx := context.Background()
	nodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("Failed to get nodes for failure detection: %v", err)
		return
	}

	now := time.Now()
	for _, node := range nodes {
		if node.ID != s.nodeID && now.Sub(node.LastSeen) > 30*time.Second {
			s.failureDetector.mu.Lock()
			s.failureDetector.failedNodes[node.ID] = now
			s.failureDetector.mu.Unlock()
			log.Printf("Detected failed node: %s", node.ID)
		}
	}
}

func (s *Scheduler) taskReassignment() {
	ticker := time.NewTicker(s.taskReassigner.reassignInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.reassignFailedTasks()
		}
	}
}

func (s *Scheduler) reassignFailedTasks() {
	if !s.isLeader {
		return
	}

	ctx := context.Background()
	failedTasks, err := s.GetTasksByStatus(types.StatusFailed)
	if err != nil {
		log.Printf("Failed to get failed tasks: %v", err)
		return
	}

	for _, task := range failedTasks {
		task.Status = types.StatusPending
		task.StartedAt = nil
		task.CompletedAt = nil
		task.WorkerID = ""
		task.Error = ""

		if err := s.storage.UpdateTask(ctx, task); err != nil {
			log.Printf("Failed to update task %s for reassignment: %v", task.ID, err)
			continue
		}

		s.queue.Add(task)
		log.Printf("Reassigned failed task %s", task.ID)
	}
}

func (s *Scheduler) backpressureMonitoring() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkBackpressure()
		}
	}
}

func (s *Scheduler) checkBackpressure() {
	queueSize := s.queue.Len()
	utilization := float64(queueSize) / float64(s.maxQueueSize)

	if utilization > 0.8 {
		log.Printf("High queue utilization: %.2f%% (%d/%d)", utilization*100, queueSize, s.maxQueueSize)
	}

	if queueSize >= s.maxQueueSize {
		log.Printf("Queue is full, rejecting new tasks")
	}
}

func (s *Scheduler) autoscaling() {
	ticker := time.NewTicker(s.autoscaler.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAutoscaling()
		}
	}
}

func (s *Scheduler) checkAutoscaling() {
	if !s.isLeader {
		return
	}

	s.autoscaler.mu.Lock()
	defer s.autoscaler.mu.Unlock()

	now := time.Now()
	if now.Sub(s.autoscaler.lastScaleTime) < s.autoscaler.cooldownPeriod {
		return
	}

	queueStats := s.GetQueueStats()
	queueUtilization := queueStats["queue_utilization"].(float64)

	workerMetrics := s.GetWorkerMetrics()
	currentWorkers := workerMetrics.WorkerCount

	if queueUtilization > s.autoscaler.scaleUpThreshold && currentWorkers < s.autoscaler.maxWorkers {
		newWorkerCount := currentWorkers + 1
		if s.scaleWorkers(newWorkerCount) {
			s.autoscaler.lastScaleTime = now
			log.Printf("Scaled up workers from %d to %d (queue utilization: %.2f%%)",
				currentWorkers, newWorkerCount, queueUtilization*100)
		}
	} else if queueUtilization < s.autoscaler.scaleDownThreshold && currentWorkers > s.autoscaler.minWorkers {
		newWorkerCount := currentWorkers - 1
		if s.scaleWorkers(newWorkerCount) {
			s.autoscaler.lastScaleTime = now
			log.Printf("Scaled down workers from %d to %d (queue utilization: %.2f%%)",
				currentWorkers, newWorkerCount, queueUtilization*100)
		}
	}
}

func (s *Scheduler) scaleWorkers(targetCount int) bool {
	if s.workerPool == nil {
		return false
	}

	// This is a simplified implementation
	// In a real implementation, you would need to properly scale the worker pool
	log.Printf("Scaling workers to %d (current implementation is simplified)", targetCount)
	return true
}

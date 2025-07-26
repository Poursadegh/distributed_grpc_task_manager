package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"task-scheduler/internal/storage"
	"task-scheduler/internal/types"
)

// IntegrationTestSuite tests multi-node scenarios
type IntegrationTestSuite struct {
	nodes   []*Scheduler
	storage storage.Storage
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewIntegrationTestSuite() *IntegrationTestSuite {
	ctx, cancel := context.WithCancel(context.Background())

	// Create Redis storage for testing
	redisClient := storage.NewRedisClient("localhost:6379", "", 0)
	storage := storage.NewRedisStorage(redisClient)

	return &IntegrationTestSuite{
		storage: storage,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (its *IntegrationTestSuite) Setup(t *testing.T) {
	// Create multiple scheduler nodes
	its.nodes = make([]*Scheduler, 3)

	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("test-node-%d", i+1)
		address := fmt.Sprintf("localhost:%d", 8080+i)

		peers := []string{}
		if i > 0 {
			peers = append(peers, fmt.Sprintf("localhost:%d", 8081))
		}

		sched := NewScheduler(nodeID, address, its.storage, peers)
		its.nodes[i] = sched

		if err := sched.Start(); err != nil {
			t.Fatalf("Failed to start node %d: %v", i+1, err)
		}
	}

	// Wait for leader election
	time.Sleep(5 * time.Second)
}

func (its *IntegrationTestSuite) Teardown() {
	its.cancel()

	for _, node := range its.nodes {
		if node != nil {
			node.Stop()
		}
	}
}

func TestMultiNodeLeaderElection(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for leader election
	time.Sleep(10 * time.Second)

	// Check that exactly one node is leader
	leaderCount := 0
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leaderCount++
		}
	}

	if leaderCount != 1 {
		t.Errorf("Expected exactly 1 leader, got %d", leaderCount)
	}

	// Find the leader
	var leader *Scheduler
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	t.Logf("Leader elected: %s", leader.GetNodeID())
}

func TestLeaderFailover(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for initial leader election
	time.Sleep(10 * time.Second)

	// Find the current leader
	var leader *Scheduler
	var leaderIndex int
	for i, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			leaderIndex = i
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	originalLeaderID := leader.GetNodeID()
	t.Logf("Original leader: %s", originalLeaderID)

	// Stop the leader
	leader.Stop()
	suite.nodes[leaderIndex] = nil

	// Wait for new leader election
	time.Sleep(15 * time.Second)

	// Check that a new leader was elected
	newLeaderCount := 0
	var newLeader *Scheduler
	for _, node := range suite.nodes {
		if node != nil && node.IsLeader() {
			newLeaderCount++
			newLeader = node
		}
	}

	if newLeaderCount != 1 {
		t.Errorf("Expected exactly 1 new leader, got %d", newLeaderCount)
	}

	if newLeader == nil {
		t.Fatal("No new leader elected")
	}

	if newLeader.GetNodeID() == originalLeaderID {
		t.Error("New leader should be different from original leader")
	}

	t.Logf("New leader elected: %s", newLeader.GetNodeID())
}

func TestTaskSubmissionAndProcessing(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for leader election
	time.Sleep(10 * time.Second)

	// Find the leader
	var leader *Scheduler
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	// Submit tasks through the leader
	tasks := []*types.Task{}
	for i := 0; i < 5; i++ {
		payload := json.RawMessage(fmt.Sprintf(`{"message": "test-task-%d"}`, i))
		task, err := leader.SubmitTask(types.PriorityHigh, payload)
		if err != nil {
			t.Fatalf("Failed to submit task %d: %v", i, err)
		}
		tasks = append(tasks, task)
	}

	// Wait for tasks to be processed
	time.Sleep(10 * time.Second)

	// Check task status
	for _, task := range tasks {
		updatedTask, err := leader.GetTask(task.ID)
		if err != nil {
			t.Errorf("Failed to get task %s: %v", task.ID, err)
			continue
		}

		if updatedTask.Status != types.StatusCompleted {
			t.Errorf("Task %s not completed, status: %s", task.ID, updatedTask.Status.String())
		}
	}
}

func TestTaskReassignmentOnFailure(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for leader election
	time.Sleep(10 * time.Second)

	// Find the leader
	var leader *Scheduler
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	// Submit a task
	payload := json.RawMessage(`{"message": "test-task-reassignment"}`)
	task, err := leader.SubmitTask(types.PriorityHigh, payload)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Wait for task to start processing
	time.Sleep(3 * time.Second)

	// Stop a non-leader node to simulate failure
	var nodeToStop *Scheduler
	for _, node := range suite.nodes {
		if node != nil && !node.IsLeader() {
			nodeToStop = node
			break
		}
	}

	if nodeToStop != nil {
		nodeToStop.Stop()
		t.Logf("Stopped node: %s", nodeToStop.GetNodeID())
	}

	// Wait for task reassignment and completion
	time.Sleep(15 * time.Second)

	// Check that task was completed
	updatedTask, err := leader.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if updatedTask.Status != types.StatusCompleted {
		t.Errorf("Task not completed after reassignment, status: %s", updatedTask.Status.String())
	}
}

func TestBackpressureHandling(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for leader election
	time.Sleep(10 * time.Second)

	// Find the leader
	var leader *Scheduler
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	// Submit many tasks to test backpressure
	successCount := 0
	for i := 0; i < 100; i++ {
		payload := json.RawMessage(fmt.Sprintf(`{"message": "backpressure-test-%d"}`, i))
		_, err := leader.SubmitTask(types.PriorityLow, payload)
		if err != nil {
			if err.Error() == "queue is full, cannot accept new tasks" {
				t.Logf("Backpressure triggered at task %d", i)
				break
			}
			t.Errorf("Unexpected error submitting task %d: %v", i, err)
		}
		successCount++
	}

	t.Logf("Successfully submitted %d tasks before backpressure", successCount)

	// Wait for some tasks to be processed
	time.Sleep(10 * time.Second)

	// Check queue stats
	stats := leader.GetQueueStats()
	totalTasks := stats["total_tasks"].(int)
	utilization := stats["queue_utilization"].(float64)

	t.Logf("Queue stats - Total: %d, Utilization: %.2f%%", totalTasks, utilization*100)

	if utilization > 1.0 {
		t.Errorf("Queue utilization should not exceed 100%%, got %.2f%%", utilization*100)
	}
}

func TestConcurrentTaskSubmission(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for leader election
	time.Sleep(10 * time.Second)

	// Find the leader
	var leader *Scheduler
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	// Submit tasks concurrently
	taskCount := 20
	results := make(chan error, taskCount)

	for i := 0; i < taskCount; i++ {
		go func(id int) {
			payload := json.RawMessage(fmt.Sprintf(`{"message": "concurrent-test-%d"}`, id))
			_, err := leader.SubmitTask(types.PriorityMedium, payload)
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < taskCount; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			t.Logf("Task %d submission failed: %v", i, err)
		}
	}

	t.Logf("Concurrent submission: %d/%d successful", successCount, taskCount)

	// Wait for processing
	time.Sleep(15 * time.Second)

	// Check final stats
	stats := leader.GetQueueStats()
	totalTasks := stats["total_tasks"].(int)

	t.Logf("Final queue size: %d", totalTasks)
}

func TestHTTPAPIEndpoints(t *testing.T) {
	suite := NewIntegrationTestSuite()
	defer suite.Teardown()

	suite.Setup(t)

	// Wait for leader election
	time.Sleep(10 * time.Second)

	// Find the leader
	var leader *Scheduler
	for _, node := range suite.nodes {
		if node.IsLeader() {
			leader = node
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	// Test health endpoint
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check returned status %d", resp.StatusCode)
	}

	// Test metrics endpoint
	resp, err = http.Get("http://localhost:8080/metrics")
	if err != nil {
		t.Fatalf("Metrics endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Metrics endpoint returned status %d", resp.StatusCode)
	}

	// Test cluster info endpoint
	resp, err = http.Get("http://localhost:8080/api/v1/cluster/info")
	if err != nil {
		t.Fatalf("Cluster info endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Cluster info endpoint returned status %d", resp.StatusCode)
	}
}

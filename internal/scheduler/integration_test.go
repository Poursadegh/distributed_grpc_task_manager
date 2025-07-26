package scheduler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"task-scheduler/internal/storage"
	"task-scheduler/internal/types"
)

type IntegrationTestSuite struct {
	schedulers []*Scheduler
	storage    storage.Storage
}

func NewIntegrationTestSuite() *IntegrationTestSuite {
	redisClient := storage.NewRedisClient("localhost:6379", "", 0)
	storage := storage.NewRedisStorage(redisClient)

	return &IntegrationTestSuite{
		storage: storage,
	}
}

func (suite *IntegrationTestSuite) SetupTest() {
	suite.schedulers = make([]*Scheduler, 4)

	for i := 0; i < 4; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)
		httpAddr := fmt.Sprintf("localhost:%d", 8080+i*2)

		var peers []string
		if i > 0 {
			peers = []string{fmt.Sprintf("localhost:%d", 8081)}
		}

		sched := NewScheduler(nodeID, httpAddr, suite.storage, peers)
		suite.schedulers[i] = sched

		if err := sched.Start(); err != nil {
			panic(fmt.Sprintf("Failed to start scheduler %d: %v", i, err))
		}
	}

	time.Sleep(2 * time.Second)
}

func (suite *IntegrationTestSuite) TearDownTest() {
	for _, sched := range suite.schedulers {
		if sched != nil {
			sched.Stop()
		}
	}
}

func TestMultiNodeLeaderElection(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	leaderCount := 0
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leaderCount++
		}
	}

	if leaderCount != 1 {
		t.Errorf("Expected exactly 1 leader, got %d", leaderCount)
	}

	var leader *Scheduler
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leader = sched
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
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	var leader *Scheduler
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leader = sched
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	originalLeaderID := leader.GetNodeID()
	t.Logf("Original leader: %s", originalLeaderID)

	leader.Stop()

	time.Sleep(3 * time.Second)

	newLeaderCount := 0
	var newLeader *Scheduler
	for _, sched := range suite.schedulers {
		if sched != leader && sched.IsLeader() {
			newLeaderCount++
			newLeader = sched
		}
	}

	if newLeaderCount != 1 {
		t.Errorf("Expected exactly 1 new leader, got %d", newLeaderCount)
	}

	if newLeader == nil {
		t.Fatal("No new leader elected")
	}

	t.Logf("New leader elected: %s", newLeader.GetNodeID())
}

func TestTaskSubmissionAndProcessing(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	var leader *Scheduler
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leader = sched
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	for i := 0; i < 5; i++ {
		payload := json.RawMessage(fmt.Sprintf(`{"test": "data-%d"}`, i))
		task, err := leader.SubmitTask(types.PriorityHigh, payload)
		if err != nil {
			t.Fatalf("Failed to submit task %d: %v", i, err)
		}

		if task == nil {
			t.Fatalf("Task %d is nil", i)
		}
	}

	time.Sleep(5 * time.Second)

	completedTasks, err := leader.GetTasksByStatus(types.StatusCompleted)
	if err != nil {
		t.Fatalf("Failed to get completed tasks: %v", err)
	}

	if len(completedTasks) < 3 {
		t.Errorf("Expected at least 3 completed tasks, got %d", len(completedTasks))
	}
}

func TestTaskReassignmentOnFailure(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	var leader *Scheduler
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leader = sched
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	payload := json.RawMessage(`{"test": "failure-test"}`)
	task, err := leader.SubmitTask(types.PriorityHigh, payload)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	time.Sleep(1 * time.Second)

	var nonLeader *Scheduler
	for _, sched := range suite.schedulers {
		if sched != leader {
			nonLeader = sched
			break
		}
	}

	if nonLeader == nil {
		t.Fatal("No non-leader found")
	}

	nonLeader.Stop()

	time.Sleep(5 * time.Second)

	completedTasks, err := leader.GetTasksByStatus(types.StatusCompleted)
	if err != nil {
		t.Fatalf("Failed to get completed tasks: %v", err)
	}

	found := false
	for _, completedTask := range completedTasks {
		if completedTask.ID == task.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Task was not reassigned and completed after node failure")
	}
}

func TestBackpressureHandling(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	var leader *Scheduler
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leader = sched
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	for i := 0; i < 100; i++ {
		payload := json.RawMessage(fmt.Sprintf(`{"test": "backpressure-%d"}`, i))
		_, err := leader.SubmitTask(types.PriorityHigh, payload)
		if err != nil {
			t.Logf("Backpressure triggered at task %d: %v", i, err)
			break
		}
	}

	time.Sleep(2 * time.Second)

	stats := leader.GetQueueStats()
	queueSize := stats["total_tasks"].(int)

	if queueSize > 0 {
		t.Logf("Queue size after backpressure: %d", queueSize)
	}
}

func TestConcurrentTaskSubmission(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	var leader *Scheduler
	for _, sched := range suite.schedulers {
		if sched.IsLeader() {
			leader = sched
			break
		}
	}

	if leader == nil {
		t.Fatal("No leader found")
	}

	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			payload := json.RawMessage(fmt.Sprintf(`{"test": "concurrent-%d"}`, id))
			_, err := leader.SubmitTask(types.PriorityHigh, payload)
			results <- err
		}(i)
	}

	for i := 0; i < 10; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent task submission %d failed: %v", i, err)
		}
	}

	time.Sleep(3 * time.Second)

	stats := leader.GetQueueStats()
	totalTasks := stats["total_tasks"].(int)

	if totalTasks < 5 {
		t.Errorf("Expected at least 5 tasks processed, got %d", totalTasks)
	}
}

func TestHTTPAPIEndpoints(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest()
	defer suite.TearDownTest()

	time.Sleep(3 * time.Second)

	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	resp, err = http.Get("http://localhost:8080/metrics")
	if err != nil {
		t.Fatalf("Metrics endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	resp, err = http.Get("http://localhost:8080/api/v1/cluster/info")
	if err != nil {
		t.Fatalf("Cluster info endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

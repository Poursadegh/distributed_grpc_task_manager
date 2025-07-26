package scheduler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"task-scheduler/internal/types"
)

type MockStorage struct {
	tasks map[string]*types.Task
	nodes map[string]*types.NodeInfo
	queue []*types.Task
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		tasks: make(map[string]*types.Task),
		nodes: make(map[string]*types.NodeInfo),
		queue: make([]*types.Task, 0),
	}
}

func (m *MockStorage) SaveTask(ctx context.Context, task *types.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *MockStorage) GetTask(ctx context.Context, id string) (*types.Task, error) {
	if task, exists := m.tasks[id]; exists {
		return task, nil
	}
	return nil, nil
}

func (m *MockStorage) UpdateTask(ctx context.Context, task *types.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *MockStorage) DeleteTask(ctx context.Context, id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *MockStorage) GetAllTasks(ctx context.Context) ([]*types.Task, error) {
	tasks := make([]*types.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (m *MockStorage) GetTasksByStatus(ctx context.Context, status types.Status) ([]*types.Task, error) {
	var tasks []*types.Task
	for _, task := range m.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *MockStorage) SaveToQueue(ctx context.Context, task *types.Task) error {
	m.queue = append(m.queue, task)
	return nil
}

func (m *MockStorage) RemoveFromQueue(ctx context.Context, taskID string) error {
	for i, task := range m.queue {
		if task.ID == taskID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockStorage) GetQueueTasks(ctx context.Context) ([]*types.Task, error) {
	return m.queue, nil
}

func (m *MockStorage) SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error {
	m.nodes[node.ID] = node
	return nil
}

func (m *MockStorage) GetNodeInfo(ctx context.Context, id string) (*types.NodeInfo, error) {
	if node, exists := m.nodes[id]; exists {
		return node, nil
	}
	return nil, nil
}

func (m *MockStorage) GetAllNodes(ctx context.Context) ([]*types.NodeInfo, error) {
	nodes := make([]*types.NodeInfo, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (m *MockStorage) DeleteNode(ctx context.Context, id string) error {
	delete(m.nodes, id)
	return nil
}

func (m *MockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

func TestScheduler_SubmitTask(t *testing.T) {
	storage := NewMockStorage()
	sched := NewScheduler("test-node", "localhost:8081", storage, nil)

	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	time.Sleep(100 * time.Millisecond)

	payload := json.RawMessage(`{"test": "data"}`)
	task, err := sched.SubmitTask(types.PriorityHigh, payload)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	if task == nil {
		t.Fatal("Expected task to be returned")
	}

	if task.Priority != types.PriorityHigh {
		t.Errorf("Expected high priority, got %s", task.Priority.String())
	}

	if task.Status != types.StatusPending {
		t.Errorf("Expected pending status, got %s", task.Status.String())
	}
}

func TestScheduler_GetTask(t *testing.T) {
	storage := NewMockStorage()
	sched := NewScheduler("test-node", "localhost:8081", storage, nil)

	// Start scheduler
	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Wait a bit for initialization
	time.Sleep(100 * time.Millisecond)

	// Submit a task
	payload := json.RawMessage(`{"test": "data"}`)
	task, err := sched.SubmitTask(types.PriorityHigh, payload)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Get the task
	retrieved, err := sched.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected task to be retrieved")
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestScheduler_GetTasksByStatus(t *testing.T) {
	storage := NewMockStorage()
	sched := NewScheduler("test-node", "localhost:8081", storage, nil)

	// Start scheduler
	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Wait a bit for initialization
	time.Sleep(100 * time.Millisecond)

	// Submit multiple tasks
	for i := 0; i < 3; i++ {
		payload := json.RawMessage(`{"test": "data"}`)
		_, err := sched.SubmitTask(types.PriorityHigh, payload)
		if err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}

	// Get pending tasks
	pendingTasks, err := sched.GetTasksByStatus(types.StatusPending)
	if err != nil {
		t.Fatalf("Failed to get pending tasks: %v", err)
	}

	if len(pendingTasks) != 3 {
		t.Errorf("Expected 3 pending tasks, got %d", len(pendingTasks))
	}
}

func TestScheduler_GetQueueStats(t *testing.T) {
	storage := NewMockStorage()
	sched := NewScheduler("test-node", "localhost:8081", storage, nil)

	// Start scheduler
	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Wait a bit for initialization
	time.Sleep(100 * time.Millisecond)

	// Submit tasks with different priorities
	payload := json.RawMessage(`{"test": "data"}`)
	sched.SubmitTask(types.PriorityHigh, payload)
	sched.SubmitTask(types.PriorityMedium, payload)
	sched.SubmitTask(types.PriorityLow, payload)

	// Get queue stats
	stats := sched.GetQueueStats()

	if stats["total_tasks"] != 3 {
		t.Errorf("Expected 3 total tasks, got %d", stats["total_tasks"])
	}

	if stats["high_priority"] != 1 {
		t.Errorf("Expected 1 high priority task, got %d", stats["high_priority"])
	}

	if stats["medium_priority"] != 1 {
		t.Errorf("Expected 1 medium priority task, got %d", stats["medium_priority"])
	}

	if stats["low_priority"] != 1 {
		t.Errorf("Expected 1 low priority task, got %d", stats["low_priority"])
	}
}

func TestScheduler_GetClusterInfo(t *testing.T) {
	storage := NewMockStorage()
	sched := NewScheduler("test-node", "localhost:8081", storage, nil)

	// Start scheduler
	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Wait a bit for initialization
	time.Sleep(100 * time.Millisecond)

	// Get cluster info
	info := sched.GetClusterInfo()

	if info == nil {
		t.Fatal("Expected cluster info to be returned")
	}

	if info.LeaderID == "" {
		t.Error("Expected leader ID to be set")
	}
}

package queue

import (
	"fmt"
	"testing"
	"time"

	"task-scheduler/internal/types"
)

func TestNewPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue()
	if pq == nil {
		t.Fatal("NewPriorityQueue returned nil")
	}

	if pq.Len() != 0 {
		t.Errorf("Expected empty queue, got length %d", pq.Len())
	}
}

func TestPriorityQueue_Add(t *testing.T) {
	pq := NewPriorityQueue()

	// Create tasks with different priorities
	highTask := &types.Task{
		ID:        "high-1",
		Priority:  types.PriorityHigh,
		CreatedAt: time.Now(),
	}

	mediumTask := &types.Task{
		ID:        "medium-1",
		Priority:  types.PriorityMedium,
		CreatedAt: time.Now(),
	}

	lowTask := &types.Task{
		ID:        "low-1",
		Priority:  types.PriorityLow,
		CreatedAt: time.Now(),
	}

	// Add tasks
	pq.Add(highTask)
	pq.Add(mediumTask)
	pq.Add(lowTask)

	if pq.Len() != 3 {
		t.Errorf("Expected queue length 3, got %d", pq.Len())
	}
}

func TestPriorityQueue_Remove(t *testing.T) {
	pq := NewPriorityQueue()

	// Create tasks with different priorities
	highTask := &types.Task{
		ID:        "high-1",
		Priority:  types.PriorityHigh,
		CreatedAt: time.Now(),
	}

	mediumTask := &types.Task{
		ID:        "medium-1",
		Priority:  types.PriorityMedium,
		CreatedAt: time.Now(),
	}

	lowTask := &types.Task{
		ID:        "low-1",
		Priority:  types.PriorityLow,
		CreatedAt: time.Now(),
	}

	// Add tasks
	pq.Add(lowTask)
	pq.Add(highTask)
	pq.Add(mediumTask)

	// Remove tasks - should get high priority first
	first := pq.Remove()
	if first == nil {
		t.Fatal("Expected to get a task, got nil")
	}
	if first.Priority != types.PriorityHigh {
		t.Errorf("Expected high priority task, got %s", first.Priority.String())
	}

	second := pq.Remove()
	if second == nil {
		t.Fatal("Expected to get a task, got nil")
	}
	if second.Priority != types.PriorityMedium {
		t.Errorf("Expected medium priority task, got %s", second.Priority.String())
	}

	third := pq.Remove()
	if third == nil {
		t.Fatal("Expected to get a task, got nil")
	}
	if third.Priority != types.PriorityLow {
		t.Errorf("Expected low priority task, got %s", third.Priority.String())
	}

	// Queue should be empty now
	if pq.Len() != 0 {
		t.Errorf("Expected empty queue, got length %d", pq.Len())
	}
}

func TestPriorityQueue_GetByID(t *testing.T) {
	pq := NewPriorityQueue()

	task := &types.Task{
		ID:        "test-1",
		Priority:  types.PriorityHigh,
		CreatedAt: time.Now(),
	}

	pq.Add(task)

	// Get by ID
	retrieved := pq.GetByID("test-1")
	if retrieved == nil {
		t.Fatal("Expected to find task, got nil")
	}

	if retrieved.ID != "test-1" {
		t.Errorf("Expected task ID 'test-1', got '%s'", retrieved.ID)
	}

	// Get non-existent task
	notFound := pq.GetByID("non-existent")
	if notFound != nil {
		t.Errorf("Expected nil for non-existent task, got %v", notFound)
	}
}

func TestPriorityQueue_Update(t *testing.T) {
	pq := NewPriorityQueue()

	task := &types.Task{
		ID:        "test-1",
		Priority:  types.PriorityHigh,
		CreatedAt: time.Now(),
	}

	pq.Add(task)

	// Update task
	task.Priority = types.PriorityLow
	pq.Update(task)

	// Remove and check priority
	removed := pq.Remove()
	if removed.Priority != types.PriorityLow {
		t.Errorf("Expected low priority after update, got %s", removed.Priority.String())
	}
}

func TestPriorityQueue_GetStats(t *testing.T) {
	pq := NewPriorityQueue()

	// Add tasks with different priorities
	pq.Add(&types.Task{ID: "high-1", Priority: types.PriorityHigh, CreatedAt: time.Now()})
	pq.Add(&types.Task{ID: "high-2", Priority: types.PriorityHigh, CreatedAt: time.Now()})
	pq.Add(&types.Task{ID: "medium-1", Priority: types.PriorityMedium, CreatedAt: time.Now()})
	pq.Add(&types.Task{ID: "low-1", Priority: types.PriorityLow, CreatedAt: time.Now()})

	stats := pq.GetStats()

	if stats["total_tasks"] != 4 {
		t.Errorf("Expected 4 total tasks, got %d", stats["total_tasks"])
	}

	if stats["high_priority"] != 2 {
		t.Errorf("Expected 2 high priority tasks, got %d", stats["high_priority"])
	}

	if stats["medium_priority"] != 1 {
		t.Errorf("Expected 1 medium priority task, got %d", stats["medium_priority"])
	}

	if stats["low_priority"] != 1 {
		t.Errorf("Expected 1 low priority task, got %d", stats["low_priority"])
	}
}

func TestPriorityQueue_ConcurrentAccess(t *testing.T) {
	pq := NewPriorityQueue()
	done := make(chan bool, 10)

	// Start multiple goroutines adding tasks
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				task := &types.Task{
					ID:        fmt.Sprintf("task-%d-%d", id, j),
					Priority:  types.PriorityHigh,
					CreatedAt: time.Now(),
				}
				pq.Add(task)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 5; i++ {
		<-done
	}

	// Check final length
	expected := 50 // 5 goroutines * 10 tasks each
	if pq.Len() != expected {
		t.Errorf("Expected %d tasks, got %d", expected, pq.Len())
	}
}

func TestPriorityQueue_FIFOForSamePriority(t *testing.T) {
	pq := NewPriorityQueue()

	// Create tasks with same priority but different creation times
	now := time.Now()
	task1 := &types.Task{
		ID:        "task-1",
		Priority:  types.PriorityHigh,
		CreatedAt: now,
	}

	task2 := &types.Task{
		ID:        "task-2",
		Priority:  types.PriorityHigh,
		CreatedAt: now.Add(time.Second), // Later creation time
	}

	// Add tasks
	pq.Add(task2) // Add later task first
	pq.Add(task1) // Add earlier task second

	// Remove tasks - should get earlier task first (FIFO for same priority)
	first := pq.Remove()
	if first.ID != "task-1" {
		t.Errorf("Expected task-1 first (FIFO), got %s", first.ID)
	}

	second := pq.Remove()
	if second.ID != "task-2" {
		t.Errorf("Expected task-2 second, got %s", second.ID)
	}
}

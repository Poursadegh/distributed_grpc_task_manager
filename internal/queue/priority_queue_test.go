package queue

import (
	"fmt"
	"testing"
	"time"

	"task-scheduler/internal/types"
)

func TestPriorityQueue_Add(t *testing.T) {
	pq := NewPriorityQueue()

	task1 := &types.Task{
		ID:        "task-1",
		Priority:  types.PriorityHigh,
		Payload:   []byte(`{"test": "data1"}`),
		CreatedAt: time.Now(),
	}

	task2 := &types.Task{
		ID:        "task-2",
		Priority:  types.PriorityMedium,
		Payload:   []byte(`{"test": "data2"}`),
		CreatedAt: time.Now(),
	}

	task3 := &types.Task{
		ID:        "task-3",
		Priority:  types.PriorityLow,
		Payload:   []byte(`{"test": "data3"}`),
		CreatedAt: time.Now(),
	}

	pq.Add(task1)
	pq.Add(task2)
	pq.Add(task3)

	if pq.Len() != 3 {
		t.Errorf("Expected queue length 3, got %d", pq.Len())
	}
}

func TestPriorityQueue_Remove(t *testing.T) {
	pq := NewPriorityQueue()

	task1 := &types.Task{
		ID:        "task-1",
		Priority:  types.PriorityLow,
		Payload:   []byte(`{"test": "data1"}`),
		CreatedAt: time.Now(),
	}

	task2 := &types.Task{
		ID:        "task-2",
		Priority:  types.PriorityHigh,
		Payload:   []byte(`{"test": "data2"}`),
		CreatedAt: time.Now(),
	}

	task3 := &types.Task{
		ID:        "task-3",
		Priority:  types.PriorityMedium,
		Payload:   []byte(`{"test": "data3"}`),
		CreatedAt: time.Now(),
	}

	pq.Add(task1)
	pq.Add(task2)
	pq.Add(task3)

	removed := pq.Remove()
	if removed == nil {
		t.Fatal("Expected to remove a task, got nil")
	}

	if removed.ID != "task-2" {
		t.Errorf("Expected to remove high priority task, got %s", removed.ID)
	}

	if pq.Len() != 2 {
		t.Errorf("Expected queue length 2, got %d", pq.Len())
	}

	removed = pq.Remove()
	if removed.ID != "task-3" {
		t.Errorf("Expected to remove medium priority task, got %s", removed.ID)
	}

	removed = pq.Remove()
	if removed.ID != "task-1" {
		t.Errorf("Expected to remove low priority task, got %s", removed.ID)
	}

	if pq.Len() != 0 {
		t.Errorf("Expected empty queue, got length %d", pq.Len())
	}
}

func TestPriorityQueue_GetByID(t *testing.T) {
	pq := NewPriorityQueue()

	task := &types.Task{
		ID:        "task-1",
		Priority:  types.PriorityHigh,
		Payload:   []byte(`{"test": "data"}`),
		CreatedAt: time.Now(),
	}

	pq.Add(task)

	found := pq.GetByID("task-1")
	if found == nil {
		t.Fatal("Expected to find task, got nil")
	}

	if found.ID != "task-1" {
		t.Errorf("Expected task ID 'task-1', got %s", found.ID)
	}

	notFound := pq.GetByID("non-existent")
	if notFound != nil {
		t.Errorf("Expected nil for non-existent task, got %v", notFound)
	}
}

func TestPriorityQueue_Update(t *testing.T) {
	pq := NewPriorityQueue()

	task := &types.Task{
		ID:        "task-1",
		Priority:  types.PriorityLow,
		Payload:   []byte(`{"test": "data"}`),
		CreatedAt: time.Now(),
	}

	pq.Add(task)

	task.Priority = types.PriorityHigh
	pq.Update(task)

	removed := pq.Remove()
	if removed.Priority != types.PriorityHigh {
		t.Errorf("Expected updated priority High, got %s", removed.Priority)
	}
}

func TestPriorityQueue_GetStats(t *testing.T) {
	pq := NewPriorityQueue()

	for i := 0; i < 5; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			Payload:   []byte(`{"test": "data"}`),
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i+5),
			Priority:  types.PriorityMedium,
			Payload:   []byte(`{"test": "data"}`),
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	for i := 0; i < 2; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i+8),
			Priority:  types.PriorityLow,
			Payload:   []byte(`{"test": "data"}`),
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	stats := pq.GetStats()

	if stats["total_tasks"].(int) != 10 {
		t.Errorf("Expected total tasks 10, got %d", stats["total_tasks"])
	}

	if stats["high_priority"].(int) != 5 {
		t.Errorf("Expected high priority tasks 5, got %d", stats["high_priority"])
	}

	if stats["medium_priority"].(int) != 3 {
		t.Errorf("Expected medium priority tasks 3, got %d", stats["medium_priority"])
	}

	if stats["low_priority"].(int) != 2 {
		t.Errorf("Expected low priority tasks 2, got %d", stats["low_priority"])
	}
}

func TestPriorityQueue_ConcurrentAccess(t *testing.T) {
	pq := NewPriorityQueue()
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				task := &types.Task{
					ID:        fmt.Sprintf("task-%d-%d", id, j),
					Priority:  types.PriorityHigh,
					Payload:   []byte(`{"test": "data"}`),
					CreatedAt: time.Now(),
				}
				pq.Add(task)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	expected := 50
	if pq.Len() != expected {
		t.Errorf("Expected queue length %d, got %d", expected, pq.Len())
	}
}

func TestPriorityQueue_FIFOForSamePriority(t *testing.T) {
	pq := NewPriorityQueue()
	now := time.Now()

	task1 := &types.Task{
		ID:        "task-1",
		Priority:  types.PriorityHigh,
		Payload:   []byte(`{"test": "data1"}`),
		CreatedAt: now,
	}

	task2 := &types.Task{
		ID:        "task-2",
		Priority:  types.PriorityHigh,
		Payload:   []byte(`{"test": "data2"}`),
		CreatedAt: now.Add(time.Second),
	}

	pq.Add(task2)
	pq.Add(task1)

	removed := pq.Remove()
	if removed.ID != "task1" {
		t.Errorf("Expected earlier task first, got %s", removed.ID)
	}

	removed = pq.Remove()
	if removed.ID != "task2" {
		t.Errorf("Expected later task second, got %s", removed.ID)
	}
}

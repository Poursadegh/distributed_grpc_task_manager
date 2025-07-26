package queue

import (
	"fmt"
	"testing"
	"time"

	"task-scheduler/internal/types"
)

func BenchmarkPriorityQueue_Add(b *testing.B) {
	pq := NewPriorityQueue()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}
}

func BenchmarkPriorityQueue_Remove(b *testing.B) {
	pq := NewPriorityQueue()

	// Pre-populate queue
	for i := 0; i < b.N; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Remove()
	}
}

func BenchmarkPriorityQueue_GetByID(b *testing.B) {
	pq := NewPriorityQueue()

	// Pre-populate queue
	for i := 0; i < 1000; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.GetByID(fmt.Sprintf("task-%d", i%1000))
	}
}

func BenchmarkPriorityQueue_Update(b *testing.B) {
	pq := NewPriorityQueue()

	// Pre-populate queue
	for i := 0; i < 1000; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i%1000),
			Priority:  types.PriorityLow,
			CreatedAt: time.Now(),
		}
		pq.Update(task)
	}
}

func BenchmarkPriorityQueue_ConcurrentAdd(b *testing.B) {
	pq := NewPriorityQueue()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := &types.Task{
				ID:        fmt.Sprintf("task-%d", i),
				Priority:  types.PriorityHigh,
				CreatedAt: time.Now(),
			}
			pq.Add(task)
			i++
		}
	})
}

func BenchmarkPriorityQueue_ConcurrentRemove(b *testing.B) {
	pq := NewPriorityQueue()

	// Pre-populate queue
	for i := 0; i < b.N; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pq.Remove()
		}
	})
}

func BenchmarkPriorityQueue_MixedOperations(b *testing.B) {
	pq := NewPriorityQueue()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				// Add task
				task := &types.Task{
					ID:        fmt.Sprintf("task-%d", i),
					Priority:  types.PriorityHigh,
					CreatedAt: time.Now(),
				}
				pq.Add(task)
			case 1:
				// Remove task
				pq.Remove()
			case 2:
				// Get by ID
				pq.GetByID(fmt.Sprintf("task-%d", i%100))
			case 3:
				// Update task
				task := &types.Task{
					ID:        fmt.Sprintf("task-%d", i%100),
					Priority:  types.PriorityLow,
					CreatedAt: time.Now(),
				}
				pq.Update(task)
			}
			i++
		}
	})
}

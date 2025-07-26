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
			Payload:   []byte(`{"test": "data"}`),
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}
}

func BenchmarkPriorityQueue_Remove(b *testing.B) {
	pq := NewPriorityQueue()

	for i := 0; i < 1000; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			Payload:   []byte(`{"test": "data"}`),
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

	for i := 0; i < 1000; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			Payload:   []byte(`{"test": "data"}`),
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.GetByID("task-500")
	}
}

func BenchmarkPriorityQueue_Update(b *testing.B) {
	pq := NewPriorityQueue()

	for i := 0; i < 1000; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			Payload:   []byte(`{"test": "data"}`),
			CreatedAt: time.Now(),
		}
		pq.Add(task)
	}

	task := &types.Task{
		ID:        "task-500",
		Priority:  types.PriorityLow,
		Payload:   []byte(`{"test": "data"}`),
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
				Payload:   []byte(`{"test": "data"}`),
				CreatedAt: time.Now(),
			}
			pq.Add(task)
			i++
		}
	})
}

func BenchmarkPriorityQueue_ConcurrentRemove(b *testing.B) {
	pq := NewPriorityQueue()

	for i := 0; i < 10000; i++ {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  types.PriorityHigh,
			Payload:   []byte(`{"test": "data"}`),
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
			if i%3 == 0 {
				task := &types.Task{
					ID:        fmt.Sprintf("task-%d", i),
					Priority:  types.PriorityHigh,
					Payload:   []byte(`{"test": "data"}`),
					CreatedAt: time.Now(),
				}
				pq.Add(task)
			} else if i%3 == 1 {
				pq.Remove()
			} else {
				pq.GetByID(fmt.Sprintf("task-%d", i))
			}
			i++
		}
	})
}

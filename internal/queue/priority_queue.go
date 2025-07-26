package queue

import (
	"container/heap"
	"sync"

	"task-scheduler/internal/types"
)

type PriorityQueue struct {
	mu      sync.RWMutex
	items   []*TaskItem
	taskMap map[string]*TaskItem
}

type TaskItem struct {
	Task     *types.Task
	Priority int
	Index    int
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items:   make([]*TaskItem, 0),
		taskMap: make(map[string]*TaskItem),
	}
	heap.Init(pq)
	return pq
}

func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	if pq.items[i].Priority != pq.items[j].Priority {
		return pq.items[i].Priority > pq.items[j].Priority
	}
	return pq.items[i].Task.CreatedAt.Before(pq.items[j].Task.CreatedAt)
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*TaskItem)
	item.Index = len(pq.items)
	pq.items = append(pq.items, item)
	pq.taskMap[item.Task.ID] = item
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	pq.items = old[0 : n-1]
	delete(pq.taskMap, item.Task.ID)
	return item
}

func (pq *PriorityQueue) Add(task *types.Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	priority := int(task.Priority)

	item := &TaskItem{
		Task:     task,
		Priority: priority,
	}

	heap.Push(pq, item)
}

func (pq *PriorityQueue) Get() *types.Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if len(pq.items) == 0 {
		return nil
	}

	return pq.items[0].Task
}

func (pq *PriorityQueue) Remove() *types.Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return nil
	}

	item := heap.Pop(pq).(*TaskItem)
	return item.Task
}

func (pq *PriorityQueue) GetByID(id string) *types.Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if item, exists := pq.taskMap[id]; exists {
		return item.Task
	}
	return nil
}

func (pq *PriorityQueue) RemoveByID(id string) *types.Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if item, exists := pq.taskMap[id]; exists {
		heap.Remove(pq, item.Index)
		return item.Task
	}
	return nil
}

func (pq *PriorityQueue) Update(task *types.Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if item, exists := pq.taskMap[task.ID]; exists {
		item.Task = task
		item.Priority = int(task.Priority)
		heap.Fix(pq, item.Index)
	}
}

func (pq *PriorityQueue) GetAll() []*types.Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	tasks := make([]*types.Task, len(pq.items))
	for i, item := range pq.items {
		tasks[i] = item.Task
	}
	return tasks
}

func (pq *PriorityQueue) GetStats() map[string]interface{} {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	stats := map[string]interface{}{
		"total_tasks":     len(pq.items),
		"high_priority":   0,
		"medium_priority": 0,
		"low_priority":    0,
	}

	for _, item := range pq.items {
		switch item.Task.Priority {
		case types.PriorityHigh:
			stats["high_priority"] = stats["high_priority"].(int) + 1
		case types.PriorityMedium:
			stats["medium_priority"] = stats["medium_priority"].(int) + 1
		case types.PriorityLow:
			stats["low_priority"] = stats["low_priority"].(int) + 1
		}
	}

	return stats
}

func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.items = make([]*TaskItem, 0)
	pq.taskMap = make(map[string]*TaskItem)
	heap.Init(pq)
}

package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"task-scheduler/internal/types"
)

// TaskProcessor defines the interface for processing tasks
type TaskProcessor interface {
	ProcessTask(ctx context.Context, task *types.Task) error
}

// WorkerPool manages a pool of workers for processing tasks
type WorkerPool struct {
	processor   TaskProcessor
	storage     Storage
	queue       Queue
	workerCount int
	workers     []*Worker
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	metrics     *WorkerMetrics
}

// Storage defines the interface for task storage operations
type Storage interface {
	UpdateTask(ctx context.Context, task *types.Task) error
	GetTask(ctx context.Context, id string) (*types.Task, error)
}

// Queue defines the interface for task queue operations
type Queue interface {
	Remove() *types.Task
	Get() *types.Task
	Add(task *types.Task)
	GetByID(id string) *types.Task
	Update(task *types.Task)
	Len() int
}

// WorkerMetrics tracks worker pool metrics
type WorkerMetrics struct {
	mu              sync.RWMutex
	TasksProcessed  int64
	TasksFailed     int64
	TasksInProgress int64
	WorkerCount     int
	QueueSize       int
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(processor TaskProcessor, storage Storage, queue Queue, workerCount int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		processor:   processor,
		storage:     storage,
		queue:       queue,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &WorkerMetrics{},
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Create workers
	wp.workers = make([]*Worker, wp.workerCount)
	for i := 0; i < wp.workerCount; i++ {
		worker := NewWorker(i, wp.processor, wp.storage, wp.queue, wp.metrics)
		wp.workers[i] = worker
	}

	// Start workers
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.Start(wp.ctx)
		}(worker)
	}

	wp.metrics.WorkerCount = wp.workerCount
	log.Printf("Started worker pool with %d workers", wp.workerCount)
	return nil
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Cancel context to signal workers to stop
	wp.cancel()

	// Wait for all workers to finish
	wp.wg.Wait()

	log.Printf("Stopped worker pool")
}

// GetMetrics returns the current metrics
func (wp *WorkerPool) GetMetrics() *WorkerMetrics {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	// Update queue size
	wp.metrics.QueueSize = wp.queue.Len()

	return wp.metrics
}

// Worker represents a single worker goroutine
type Worker struct {
	id         int
	processor  TaskProcessor
	storage    Storage
	queue      Queue
	metrics    *WorkerMetrics
	processing bool
}

// NewWorker creates a new worker
func NewWorker(id int, processor TaskProcessor, storage Storage, queue Queue, metrics *WorkerMetrics) *Worker {
	return &Worker{
		id:        id,
		processor: processor,
		storage:   storage,
		queue:     queue,
		metrics:   metrics,
	}
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	log.Printf("Worker %d started", w.id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping", w.id)
			return
		default:
			// Try to get a task from the queue
			task := w.queue.Remove()
			if task == nil {
				// No tasks available, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Process the task
			w.processTask(ctx, task)
		}
	}
}

// processTask processes a single task
func (w *Worker) processTask(ctx context.Context, task *types.Task) {
	w.metrics.mu.Lock()
	w.metrics.TasksInProgress++
	w.processing = true
	w.metrics.mu.Unlock()

	defer func() {
		w.metrics.mu.Lock()
		w.metrics.TasksInProgress--
		w.processing = false
		w.metrics.mu.Unlock()
	}()

	// Update task status to running
	now := time.Now()
	task.Status = types.StatusRunning
	task.StartedAt = &now
	task.WorkerID = fmt.Sprintf("worker-%d", w.id)

	// Save task status
	if err := w.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Worker %d: failed to update task %s status: %v", w.id, task.ID, err)
	}

	log.Printf("Worker %d: processing task %s (priority: %s)", w.id, task.ID, task.Priority.String())

	// Process the task
	err := w.processor.ProcessTask(ctx, task)

	// Update task status based on result
	if err != nil {
		task.Status = types.StatusFailed
		task.Error = err.Error()
		log.Printf("Worker %d: task %s failed: %v", w.id, task.ID, err)

		w.metrics.mu.Lock()
		w.metrics.TasksFailed++
		w.metrics.mu.Unlock()
	} else {
		task.Status = types.StatusCompleted
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		log.Printf("Worker %d: task %s completed successfully", w.id, task.ID)

		w.metrics.mu.Lock()
		w.metrics.TasksProcessed++
		w.metrics.mu.Unlock()
	}

	// Save final task status
	if err := w.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Worker %d: failed to update final task %s status: %v", w.id, task.ID, err)
	}
}

// IsProcessing returns true if the worker is currently processing a task
func (w *Worker) IsProcessing() bool {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()
	return w.processing
}

// GetMetrics returns the current metrics
func (w *Worker) GetMetrics() map[string]interface{} {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()

	return map[string]interface{}{
		"worker_id":         w.id,
		"processing":        w.processing,
		"tasks_processed":   w.metrics.TasksProcessed,
		"tasks_failed":      w.metrics.TasksFailed,
		"tasks_in_progress": w.metrics.TasksInProgress,
	}
}

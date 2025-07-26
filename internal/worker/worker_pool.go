package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"task-scheduler/internal/types"
)

type TaskProcessor interface {
	ProcessTask(ctx context.Context, task *types.Task) error
}

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

type Storage interface {
	UpdateTask(ctx context.Context, task *types.Task) error
	GetTask(ctx context.Context, id string) (*types.Task, error)
}

type Queue interface {
	Remove() *types.Task
	Get() *types.Task
	Add(task *types.Task)
	GetByID(id string) *types.Task
	Update(task *types.Task)
	Len() int
}

type WorkerMetrics struct {
	mu              sync.RWMutex
	TasksProcessed  int64
	TasksFailed     int64
	TasksInProgress int64
	WorkerCount     int
	QueueSize       int
}

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

func (wp *WorkerPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.workers = make([]*Worker, wp.workerCount)
	for i := 0; i < wp.workerCount; i++ {
		worker := NewWorker(i, wp.processor, wp.storage, wp.queue, wp.metrics)
		wp.workers[i] = worker
	}

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

func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.cancel()

	wp.wg.Wait()

	log.Printf("Stopped worker pool")
}

func (wp *WorkerPool) GetMetrics() *WorkerMetrics {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	wp.metrics.QueueSize = wp.queue.Len()

	return wp.metrics
}

type Worker struct {
	id         int
	processor  TaskProcessor
	storage    Storage
	queue      Queue
	metrics    *WorkerMetrics
	processing bool
}

func NewWorker(id int, processor TaskProcessor, storage Storage, queue Queue, metrics *WorkerMetrics) *Worker {
	return &Worker{
		id:        id,
		processor: processor,
		storage:   storage,
		queue:     queue,
		metrics:   metrics,
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Printf("Worker %d started", w.id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping", w.id)
			return
		default:
			task := w.queue.Remove()
			if task == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			w.processTask(ctx, task)
		}
	}
}

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

	now := time.Now()
	task.Status = types.StatusRunning
	task.StartedAt = &now
	task.WorkerID = fmt.Sprintf("worker-%d", w.id)

	if err := w.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Worker %d: failed to update task %s status: %v", w.id, task.ID, err)
	}

	log.Printf("Worker %d: processing task %s (priority: %s)", w.id, task.ID, task.Priority.String())

	err := w.processor.ProcessTask(ctx, task)

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

	if err := w.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Worker %d: failed to update final task %s status: %v", w.id, task.ID, err)
	}
}

func (w *Worker) IsProcessing() bool {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()
	return w.processing
}

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

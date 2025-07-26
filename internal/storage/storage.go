package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"task-scheduler/internal/types"
)

// Storage defines the interface for task persistence
type Storage interface {
	// Task operations
	SaveTask(ctx context.Context, task *types.Task) error
	GetTask(ctx context.Context, id string) (*types.Task, error)
	UpdateTask(ctx context.Context, task *types.Task) error
	DeleteTask(ctx context.Context, id string) error
	GetAllTasks(ctx context.Context) ([]*types.Task, error)
	GetTasksByStatus(ctx context.Context, status types.Status) ([]*types.Task, error)

	// Queue operations
	SaveToQueue(ctx context.Context, task *types.Task) error
	RemoveFromQueue(ctx context.Context, taskID string) error
	GetQueueTasks(ctx context.Context) ([]*types.Task, error)

	// Cluster operations
	SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error
	GetNodeInfo(ctx context.Context, id string) (*types.NodeInfo, error)
	GetAllNodes(ctx context.Context) ([]*types.NodeInfo, error)
	DeleteNode(ctx context.Context, id string) error

	// Health check
	Ping(ctx context.Context) error
	Close() error
}

// RedisStorage implements Storage interface using Redis
type RedisStorage struct {
	client RedisClient
}

// RedisClient defines the interface for Redis operations
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPop(ctx context.Context, key string) (string, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	LRem(ctx context.Context, key string, count int64, value interface{}) error
	Ping(ctx context.Context) error
	Close() error
}

// NewRedisStorage creates a new Redis storage instance
func NewRedisStorage(client RedisClient) *RedisStorage {
	return &RedisStorage{
		client: client,
	}
}

// key generators
func taskKey(id string) string {
	return fmt.Sprintf("task:%s", id)
}

func queueKey() string {
	return "queue:pending"
}

func nodeKey(id string) string {
	return fmt.Sprintf("node:%s", id)
}

func nodesKey() string {
	return "nodes:all"
}

// SaveTask saves a task to storage
func (r *RedisStorage) SaveTask(ctx context.Context, task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	key := taskKey(task.ID)
	return r.client.Set(ctx, key, string(data), 0)
}

// GetTask retrieves a task by ID
func (r *RedisStorage) GetTask(ctx context.Context, id string) (*types.Task, error) {
	key := taskKey(id)
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task types.Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (r *RedisStorage) UpdateTask(ctx context.Context, task *types.Task) error {
	return r.SaveTask(ctx, task)
}

// DeleteTask removes a task from storage
func (r *RedisStorage) DeleteTask(ctx context.Context, id string) error {
	key := taskKey(id)
	return r.client.Del(ctx, key)
}

// GetAllTasks retrieves all tasks
func (r *RedisStorage) GetAllTasks(ctx context.Context) ([]*types.Task, error) {
	pattern := "task:*"
	keys, err := r.client.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get task keys: %w", err)
	}

	var tasks []*types.Task
	for _, key := range keys {
		data, err := r.client.Get(ctx, key)
		if err != nil {
			continue // skip failed tasks
		}

		var task types.Task
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue // skip invalid tasks
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetTasksByStatus retrieves tasks by status
func (r *RedisStorage) GetTasksByStatus(ctx context.Context, status types.Status) ([]*types.Task, error) {
	allTasks, err := r.GetAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	var filteredTasks []*types.Task
	for _, task := range allTasks {
		if task.Status == status {
			filteredTasks = append(filteredTasks, task)
		}
	}

	return filteredTasks, nil
}

// SaveToQueue adds a task to the pending queue
func (r *RedisStorage) SaveToQueue(ctx context.Context, task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task for queue: %w", err)
	}

	key := queueKey()
	return r.client.LPush(ctx, key, string(data))
}

// RemoveFromQueue removes a task from the pending queue
func (r *RedisStorage) RemoveFromQueue(ctx context.Context, taskID string) error {
	// This is a simplified implementation
	// In a real implementation, you'd need to scan the list and remove the specific task
	key := queueKey()
	tasks, err := r.client.LRange(ctx, key, 0, -1)
	if err != nil {
		return fmt.Errorf("failed to get queue tasks: %w", err)
	}

	for _, taskData := range tasks {
		var task types.Task
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			continue
		}

		if task.ID == taskID {
			return r.client.LRem(ctx, key, 1, taskData)
		}
	}

	return nil
}

// GetQueueTasks retrieves all tasks in the pending queue
func (r *RedisStorage) GetQueueTasks(ctx context.Context) ([]*types.Task, error) {
	key := queueKey()
	tasksData, err := r.client.LRange(ctx, key, 0, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue tasks: %w", err)
	}

	var tasks []*types.Task
	for _, taskData := range tasksData {
		var task types.Task
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			continue // skip invalid tasks
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// SaveNodeInfo saves node information
func (r *RedisStorage) SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error {
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	key := nodeKey(node.ID)
	return r.client.Set(ctx, key, string(data), 30*time.Second) // 30s TTL for node info
}

// GetNodeInfo retrieves node information
func (r *RedisStorage) GetNodeInfo(ctx context.Context, id string) (*types.NodeInfo, error) {
	key := nodeKey(id)
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	var node types.NodeInfo
	if err := json.Unmarshal([]byte(data), &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	return &node, nil
}

// GetAllNodes retrieves all node information
func (r *RedisStorage) GetAllNodes(ctx context.Context) ([]*types.NodeInfo, error) {
	pattern := "node:*"
	keys, err := r.client.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get node keys: %w", err)
	}

	var nodes []*types.NodeInfo
	for _, key := range keys {
		data, err := r.client.Get(ctx, key)
		if err != nil {
			continue // skip failed nodes
		}

		var node types.NodeInfo
		if err := json.Unmarshal([]byte(data), &node); err != nil {
			continue // skip invalid nodes
		}

		nodes = append(nodes, &node)
	}

	return nodes, nil
}

// DeleteNode removes node information
func (r *RedisStorage) DeleteNode(ctx context.Context, id string) error {
	key := nodeKey(id)
	return r.client.Del(ctx, key)
}

// Ping checks if the storage is available
func (r *RedisStorage) Ping(ctx context.Context) error {
	return r.client.Ping(ctx)
}

// Close closes the storage connection
func (r *RedisStorage) Close() error {
	return r.client.Close()
}

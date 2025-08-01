package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"task-scheduler/internal/types"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrNodeNotFound = errors.New("node not found")
)

type Storage interface {
	SaveTask(ctx context.Context, task *types.Task) error
	GetTask(ctx context.Context, id string) (*types.Task, error)
	UpdateTask(ctx context.Context, task *types.Task) error
	DeleteTask(ctx context.Context, id string) error
	GetAllTasks(ctx context.Context) ([]*types.Task, error)
	GetTasksByStatus(ctx context.Context, status types.Status) ([]*types.Task, error)

	SaveToQueue(ctx context.Context, task *types.Task) error
	RemoveFromQueue(ctx context.Context, taskID string) error
	GetQueueTasks(ctx context.Context) ([]*types.Task, error)

	SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error
	GetNodeInfo(ctx context.Context, id string) (*types.NodeInfo, error)
	GetAllNodes(ctx context.Context) ([]*types.NodeInfo, error)
	DeleteNode(ctx context.Context, id string) error

	Ping(ctx context.Context) error
	Close() error
}

type RedisStorage struct {
	client RedisClient
}

func NewRedisStorage(client RedisClient) *RedisStorage {
	return &RedisStorage{
		client: client,
	}
}

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

func (r *RedisStorage) SaveTask(ctx context.Context, task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	key := taskKey(task.ID)
	return r.client.Set(ctx, key, string(data), 0)
}

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

func (r *RedisStorage) UpdateTask(ctx context.Context, task *types.Task) error {
	return r.SaveTask(ctx, task)
}

func (r *RedisStorage) DeleteTask(ctx context.Context, id string) error {
	key := taskKey(id)
	return r.client.Del(ctx, key)
}

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
			continue
		}

		var task types.Task
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

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

func (r *RedisStorage) SaveToQueue(ctx context.Context, task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task for queue: %w", err)
	}

	key := queueKey()
	return r.client.LPush(ctx, key, string(data))
}

func (r *RedisStorage) RemoveFromQueue(ctx context.Context, taskID string) error {
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
			continue
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (r *RedisStorage) SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error {
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	key := nodeKey(node.ID)
	return r.client.Set(ctx, key, string(data), 30*time.Second)
}

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
			continue
		}

		var node types.NodeInfo
		if err := json.Unmarshal([]byte(data), &node); err != nil {
			continue
		}

		nodes = append(nodes, &node)
	}

	return nodes, nil
}

func (r *RedisStorage) DeleteNode(ctx context.Context, id string) error {
	key := nodeKey(id)
	return r.client.Del(ctx, key)
}

func (r *RedisStorage) Ping(ctx context.Context) error {
	return r.client.Ping(ctx)
}

func (r *RedisStorage) Close() error {
	return r.client.Close()
}

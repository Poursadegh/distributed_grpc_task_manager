package storage

import (
	"context"
	"fmt"

	"task-scheduler/internal/types"
)

type HybridStorage struct {
	redis    *RedisStorage
	postgres *PostgresStorage
}

func NewHybridStorage(redisClient RedisClient, postgresDSN string) (*HybridStorage, error) {
	redisStorage := NewRedisStorage(redisClient)

	postgresStorage, err := NewPostgresStorage(postgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL storage: %w", err)
	}

	return &HybridStorage{
		redis:    redisStorage,
		postgres: postgresStorage,
	}, nil
}

func (h *HybridStorage) SaveTask(ctx context.Context, task *types.Task) error {
	if err := h.redis.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save task to Redis: %w", err)
	}

	if err := h.postgres.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save task to PostgreSQL: %w", err)
	}

	return nil
}

func (h *HybridStorage) GetTask(ctx context.Context, id string) (*types.Task, error) {
	task, err := h.redis.GetTask(ctx, id)
	if err == nil && task != nil {
		return task, nil
	}

	task, err = h.postgres.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	if task != nil {
		if err := h.redis.SaveTask(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to cache task in Redis: %w", err)
		}
	}

	return task, nil
}

func (h *HybridStorage) UpdateTask(ctx context.Context, task *types.Task) error {
	if err := h.redis.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task in Redis: %w", err)
	}

	if err := h.postgres.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task in PostgreSQL: %w", err)
	}

	return nil
}

func (h *HybridStorage) DeleteTask(ctx context.Context, id string) error {
	if err := h.redis.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task from Redis: %w", err)
	}

	if err := h.postgres.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task from PostgreSQL: %w", err)
	}

	return nil
}

func (h *HybridStorage) GetAllTasks(ctx context.Context) ([]*types.Task, error) {
	return h.postgres.GetAllTasks(ctx)
}

func (h *HybridStorage) GetTasksByStatus(ctx context.Context, status types.Status) ([]*types.Task, error) {
	return h.postgres.GetTasksByStatus(ctx, status)
}

func (h *HybridStorage) SaveToQueue(ctx context.Context, task *types.Task) error {
	if err := h.redis.SaveToQueue(ctx, task); err != nil {
		return fmt.Errorf("failed to save task to Redis queue: %w", err)
	}

	if err := h.postgres.SaveToQueue(ctx, task); err != nil {
		return fmt.Errorf("failed to save task to PostgreSQL queue: %w", err)
	}

	return nil
}

func (h *HybridStorage) RemoveFromQueue(ctx context.Context, taskID string) error {
	if err := h.redis.RemoveFromQueue(ctx, taskID); err != nil {
		return fmt.Errorf("failed to remove task from Redis queue: %w", err)
	}

	if err := h.postgres.RemoveFromQueue(ctx, taskID); err != nil {
		return fmt.Errorf("failed to remove task from PostgreSQL queue: %w", err)
	}

	return nil
}

func (h *HybridStorage) GetQueueTasks(ctx context.Context) ([]*types.Task, error) {
	return h.redis.GetQueueTasks(ctx)
}

func (h *HybridStorage) SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error {
	if err := h.redis.SaveNodeInfo(ctx, node); err != nil {
		return fmt.Errorf("failed to save node info to Redis: %w", err)
	}

	if err := h.postgres.SaveNodeInfo(ctx, node); err != nil {
		return fmt.Errorf("failed to save node info to PostgreSQL: %w", err)
	}

	return nil
}

func (h *HybridStorage) GetNodeInfo(ctx context.Context, id string) (*types.NodeInfo, error) {
	node, err := h.redis.GetNodeInfo(ctx, id)
	if err == nil && node != nil {
		return node, nil
	}

	node, err = h.postgres.GetNodeInfo(ctx, id)
	if err != nil {
		return nil, err
	}

	if node != nil {
		if err := h.redis.SaveNodeInfo(ctx, node); err != nil {
			return nil, fmt.Errorf("failed to cache node info in Redis: %w", err)
		}
	}

	return node, nil
}

func (h *HybridStorage) GetAllNodes(ctx context.Context) ([]*types.NodeInfo, error) {
	return h.postgres.GetAllNodes(ctx)
}

func (h *HybridStorage) DeleteNode(ctx context.Context, id string) error {
	if err := h.redis.DeleteNode(ctx, id); err != nil {
		return fmt.Errorf("failed to delete node from Redis: %w", err)
	}

	if err := h.postgres.DeleteNode(ctx, id); err != nil {
		return fmt.Errorf("failed to delete node from PostgreSQL: %w", err)
	}

	return nil
}

func (h *HybridStorage) Ping(ctx context.Context) error {
	if err := h.redis.Ping(ctx); err != nil {
		return fmt.Errorf("Redis ping failed: %w", err)
	}

	if err := h.postgres.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL ping failed: %w", err)
	}

	return nil
}

func (h *HybridStorage) Close() error {
	if err := h.redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	if err := h.postgres.Close(); err != nil {
		return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
	}

	return nil
}

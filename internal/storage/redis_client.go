package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClientImpl implements RedisClient interface using go-redis
type RedisClientImpl struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(addr string, password string, db int) *RedisClientImpl {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisClientImpl{
		client: client,
	}
}

// Set sets a key-value pair
func (r *RedisClientImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (r *RedisClientImpl) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Del deletes keys
func (r *RedisClientImpl) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist
func (r *RedisClientImpl) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Keys gets keys matching pattern
func (r *RedisClientImpl) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

// LPush pushes values to the left of a list
func (r *RedisClientImpl) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

// RPop pops a value from the right of a list
func (r *RedisClientImpl) RPop(ctx context.Context, key string) (string, error) {
	return r.client.RPop(ctx, key).Result()
}

// LRange gets a range of elements from a list
func (r *RedisClientImpl) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

// LRem removes elements from a list
func (r *RedisClientImpl) LRem(ctx context.Context, key string, count int64, value interface{}) error {
	return r.client.LRem(ctx, key, count, value).Err()
}

// Ping pings the Redis server
func (r *RedisClientImpl) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisClientImpl) Close() error {
	return r.client.Close()
}

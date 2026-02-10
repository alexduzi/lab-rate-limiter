package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(addr, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStorage{client: client}, nil
}

func (r *RedisStorage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := r.client.Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return incr.Val(), nil
}

func (r *RedisStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockedKey := key + ":blocked"

	exists, err := r.client.Exists(ctx, blockedKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if blocked: %w", err)
	}

	return exists > 0, nil
}

func (r *RedisStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	blockedKey := key + ":blocked"

	err := r.client.Set(ctx, blockedKey, time.Now().Add(duration).Unix(), duration).Err()
	if err != nil {
		return fmt.Errorf("failed to block: %w", err)
	}

	return nil
}

func (r *RedisStorage) Reset(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset: %w", err)
	}

	return nil
}

func (r *RedisStorage) Close() error {
	return r.client.Close()
}

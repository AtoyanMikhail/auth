package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AtoyanMikhail/auth/internal/config"
	"github.com/AtoyanMikhail/auth/internal/logger"
	"github.com/redis/go-redis/v9"
)

// Key prefixes
const (
	TokenBlacklistPrefix = "blacklist:token:"
	UserBlacklistPrefix  = "blacklist:user:"
	IPAttemptPrefix      = "ip_attempt:"
)

type redisCache struct {
	client *redis.Client
	logger logger.Logger
	cfg    config.RedisConfig
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg config.RedisConfig, l logger.Logger) (Cache, error) {
	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	l.Info("Redis connection established",
		logger.String("addr", cfg.Addr),
		logger.Int("db", cfg.DB))

	return &redisCache{
		client: client,
		logger: l,
		cfg:    cfg,
	}, nil
}

// Set saves value by key with TTL
func (r *redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var data string
	switch v := value.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		data = string(jsonData)
	}

	err := r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		r.logger.Error("Failed to set cache value",
			logger.String("key", key),
			logger.Error(err))
		return fmt.Errorf("failed to set cache value: %w", err)
	}

	return nil
}

// Get gets value by key
func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key not found: %s", key)
		}
		r.logger.Error("Failed to get cache value",
			logger.String("key", key),
			logger.Error(err))
		return "", fmt.Errorf("failed to get cache value: %w", err)
	}

	return val, nil
}

// Delete deletes value by key
func (r *redisCache) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error("Failed to delete cache value",
			logger.String("key", key),
			logger.Error(err))
		return fmt.Errorf("failed to delete cache value: %w", err)
	}

	return nil
}

// Exists checks whether the key exists
func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to check key existence",
			logger.String("key", key),
			logger.Error(err))
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return count > 0, nil
}

// SetNX sets value only if key doesn't exist
func (r *redisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	var data string
	switch v := value.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
		data = string(jsonData)
	}

	success, err := r.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		r.logger.Error("Failed to set cache value with SetNX",
			logger.String("key", key),
			logger.Error(err))
		return false, fmt.Errorf("failed to set cache value with SetNX: %w", err)
	}

	return success, nil
}

// Increment increments integer value in cache by 1
func (r *redisCache) Increment(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to increment cache value",
			logger.String("key", key),
			logger.Error(err))
		return 0, fmt.Errorf("failed to increment cache value: %w", err)
	}

	return val, nil
}

// IncrementWithTTL increments value and sets TTL if the key is new
func (r *redisCache) IncrementWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	expireCmd := pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Failed to increment with TTL",
			logger.String("key", key),
			logger.Error(err))
		return 0, fmt.Errorf("failed to increment with TTL: %w", err)
	}

	val, err := incrCmd.Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get increment result: %w", err)
	}

	// Check if TTL was set successfully
	if err := expireCmd.Err(); err != nil {
		r.logger.Warn("Failed to set TTL after increment",
			logger.String("key", key),
			logger.Error(err))
	}

	return val, nil
}

// Close closes redis connection
func (r *redisCache) Close() error {
	err := r.client.Close()
	if err != nil {
		r.logger.Error("Failed to close Redis connection", logger.Error(err))
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	r.logger.Info("Redis connection closed")
	return nil
}

// Ping return error if no connection to redis
func (r *redisCache) Ping(ctx context.Context) error {
	err := r.client.Ping(ctx).Err()
	if err != nil {
		r.logger.Error("Redis ping failed", logger.Error(err))
		return fmt.Errorf("Redis ping failed: %w", err)
	}

	return nil
}

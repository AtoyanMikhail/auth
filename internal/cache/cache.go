package cache

import (
	"context"
	"time"
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	Increment(ctx context.Context, key string) (int64, error)
	IncrementWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Close() error
	Ping(ctx context.Context) error
}

type JWTCache interface {
	BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error)
	LogIPAttempt(ctx context.Context, userID, ipAddress string) error
	GetIPAttempts(ctx context.Context, userID, ipAddress string) (int64, error)
	BlacklistUser(ctx context.Context, userID string, duration time.Duration) error
	IsUserBlacklisted(ctx context.Context, userID string) (bool, error)
}
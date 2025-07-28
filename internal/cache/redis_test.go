package cache

import (
	"context"
	"testing"
	"time"

	"github.com/AtoyanMikhail/auth/internal/config"
	"github.com/AtoyanMikhail/auth/internal/logger"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Info(msg string, fields ...logger.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...logger.Field)   {}
func (m *mockLogger) Error(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Fatal(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Panic(msg string, fields ...logger.Field)  {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                               { return nil }
func (m *mockLogger) SetLevel(level logger.Level)               {}

// Test setup helper
func SetupTestRedis(t *testing.T) (*redisCache, *miniredis.Miniredis, func()) {
	// mock redis
	mr := miniredis.RunT(t)

	cfg := config.RedisConfig{
		Addr:     mr.Addr(),
		Password: "",
		DB:       0,
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	cache := &redisCache{
		client: client,
		logger: &mockLogger{},
		cfg:    cfg,
	}

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return cache, mr, cleanup
}

func TestRedisCache_Set(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		value   interface{}
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "set string value",
			key:     "test:string",
			value:   "hello world",
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "set byte slice value",
			key:     "test:bytes",
			value:   []byte("hello bytes"),
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "set struct value",
			key:     "test:struct",
			value:   struct{ Name string }{Name: "test"},
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "set with zero ttl",
			key:     "test:zero_ttl",
			value:   "persistent",
			ttl:     0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.Set(ctx, tt.key, tt.value, tt.ttl)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify the value was set
				exists, err := cache.Exists(ctx, tt.key)
				assert.NoError(t, err)
				assert.True(t, exists)
			}
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	testKey := "test:get"
	testValue := "test value"
	err := cache.Set(ctx, testKey, testValue, time.Minute)
	require.NoError(t, err)

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "get existing key",
			key:       testKey,
			wantValue: testValue,
			wantErr:   false,
		},
		{
			name:    "get non-existing key",
			key:     "test:nonexistent",
			wantErr: true,
			errMsg:  "key not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := cache.Get(ctx, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}

func TestRedisCache_Delete(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	testKey := "test:delete"
	err := cache.Set(ctx, testKey, "value", time.Minute)
	require.NoError(t, err)

	// Delete 
	err = cache.Delete(ctx, testKey)
	assert.NoError(t, err)

	// Verify it's deleted
	exists, err := cache.Exists(ctx, testKey)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Deleting non-existent key should not return an error
	err = cache.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestRedisCache_Exists(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	// Test non-existing key
	exists, err := cache.Exists(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Set a key and test it exists
	testKey := "test:exists"
	err = cache.Set(ctx, testKey, "value", time.Minute)
	require.NoError(t, err)

	exists, err = cache.Exists(ctx, testKey)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRedisCache_SetNX(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	testKey := "test:setnx"

	tests := []struct {
		name        string
		setupFn     func()
		key         string
		value       interface{}
		ttl         time.Duration
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "set new key",
			setupFn:     func() {},
			key:         testKey,
			value:       "new value",
			ttl:         time.Minute,
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "set existing key",
			setupFn: func() {
				cache.Set(ctx, testKey+"2", "existing", time.Minute)
			},
			key:         testKey + "2",
			value:       "new value",
			ttl:         time.Minute,
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name:        "set struct value",
			setupFn:     func() {},
			key:         testKey + "3",
			value:       struct{ Name string }{Name: "test"},
			ttl:         time.Minute,
			wantSuccess: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()

			success, err := cache.SetNX(ctx, tt.key, tt.value, tt.ttl)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSuccess, success)
			}
		})
	}
}

func TestRedisCache_Increment(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	testKey := "test:incr"

	// First increment should return 1
	val, err := cache.Increment(ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Second should return 2
	val, err = cache.Increment(ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)

	// Third should return 3
	val, err = cache.Increment(ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), val)
}

func TestRedisCache_IncrementWithTTL(t *testing.T) {
	cache, mr, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	testKey := "test:incr_ttl"
	ttl := time.Second

	// First increment with TTL
	val, err := cache.IncrementWithTTL(ctx, testKey, ttl)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Check that TTL is set
	exists, err := cache.Exists(ctx, testKey)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Fast forward time to expire the key
	mr.FastForward(ttl + time.Millisecond)

	// Key should be expired
	exists, err = cache.Exists(ctx, testKey)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisCache_Ping(t *testing.T) {
	cache, _, cleanup := SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	err := cache.Ping(ctx)
	assert.NoError(t, err)
}

func TestRedisCache_Close(t *testing.T) {
	cache, mr, _ := SetupTestRedis(t)

	err := cache.Close()
	assert.NoError(t, err)

	mr.Close()
}

func TestNewRedisCache(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	tests := []struct {
		name    string
		cfg     config.RedisConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: config.RedisConfig{
				Addr:     mr.Addr(),
				Password: "",
				DB:       0,
			},
			wantErr: false,
		},
		{
			name: "invalid address",
			cfg: config.RedisConfig{
				Addr:     "invalid:99999",
				Password: "",
				DB:       0,
			},
			wantErr: true,
			errMsg:  "failed to connect to Redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewRedisCache(tt.cfg, &mockLogger{})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cache)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cache)

				// Clean up
				if cache != nil {
					cache.Close()
				}
			}
		})
	}
}

package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Cache for testing JWT cache
type mockCache struct {
	mock.Mock
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *mockCache) Increment(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCache) IncrementWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	args := m.Called(ctx, key, ttl)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func SetupJWTCache(t *testing.T) (*jwtCache, *mockCache) {
	mockCacheImpl := &mockCache{}
	jwtCache := &jwtCache{
		cache:  mockCacheImpl,
		logger: &mockLogger{},
	}
	return jwtCache, mockCacheImpl
}

func TestJWTCache_BlacklistToken(t *testing.T) {
	jwtCache, mockCacheImpl := SetupJWTCache(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		tokenID   string
		expiresAt time.Time
		setupMock func(*mockCache)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful blacklist",
			tokenID:   "token123",
			expiresAt: time.Now().Add(time.Hour),
			setupMock: func(m *mockCache) {
				expectedKey := TokenBlacklistPrefix + "token123"
				m.On("Set", ctx, expectedKey, "blacklisted", mock.AnythingOfType("time.Duration")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "expired token not blacklisted",
			tokenID:   "expired_token",
			expiresAt: time.Now().Add(-time.Hour),
			setupMock: func(m *mockCache) {
				// No cache call should be made for expired token
			},
			wantErr: false,
		},
		{
			name:      "cache error",
			tokenID:   "token456",
			expiresAt: time.Now().Add(time.Hour),
			setupMock: func(m *mockCache) {
				expectedKey := TokenBlacklistPrefix + "token456"
				m.On("Set", ctx, expectedKey, "blacklisted", mock.AnythingOfType("time.Duration")).Return(fmt.Errorf("cache error"))
			},
			wantErr: true,
			errMsg:  "failed to blacklist token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockCacheImpl.ExpectedCalls = nil
			tt.setupMock(mockCacheImpl)

			err := jwtCache.BlacklistToken(ctx, tt.tokenID, tt.expiresAt)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockCacheImpl.AssertExpectations(t)
		})
	}
}

func TestJWTCache_IsTokenBlacklisted(t *testing.T) {
	jwtCache, mockCacheImpl := SetupJWTCache(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		tokenID    string
		setupMock  func(*mockCache)
		wantResult bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:    "token is blacklisted",
			tokenID: "blacklisted_token",
			setupMock: func(m *mockCache) {
				expectedKey := TokenBlacklistPrefix + "blacklisted_token"
				m.On("Exists", ctx, expectedKey).Return(true, nil)
			},
			wantResult: true,
			wantErr:    false,
		},
		{
			name:    "token is not blacklisted",
			tokenID: "clean_token",
			setupMock: func(m *mockCache) {
				expectedKey := TokenBlacklistPrefix + "clean_token"
				m.On("Exists", ctx, expectedKey).Return(false, nil)
			},
			wantResult: false,
			wantErr:    false,
		},
		{
			name:    "cache error",
			tokenID: "error_token",
			setupMock: func(m *mockCache) {
				expectedKey := TokenBlacklistPrefix + "error_token"
				m.On("Exists", ctx, expectedKey).Return(false, fmt.Errorf("cache error"))
			},
			wantResult: false,
			wantErr:    true,
			errMsg:     "failed to check token blacklist status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCacheImpl.ExpectedCalls = nil
			tt.setupMock(mockCacheImpl)

			result, err := jwtCache.IsTokenBlacklisted(ctx, tt.tokenID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}

			mockCacheImpl.AssertExpectations(t)
		})
	}
}

func TestJWTCache_LogIPAttempt(t *testing.T) {
	jwtCache, mockCacheImpl := SetupJWTCache(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    string
		ipAddress string
		setupMock func(*mockCache)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful log",
			userID:    "user123",
			ipAddress: "192.168.1.1",
			setupMock: func(m *mockCache) {
				expectedKey := fmt.Sprintf("%suser123:192.168.1.1", IPAttemptPrefix)
				m.On("IncrementWithTTL", ctx, expectedKey, 24*time.Hour).Return(int64(1), nil)
			},
			wantErr: false,
		},
		{
			name:      "cache error",
			userID:    "user456",
			ipAddress: "192.168.1.2",
			setupMock: func(m *mockCache) {
				expectedKey := fmt.Sprintf("%suser456:192.168.1.2", IPAttemptPrefix)
				m.On("IncrementWithTTL", ctx, expectedKey, 24*time.Hour).Return(int64(0), fmt.Errorf("cache error"))
			},
			wantErr: true,
			errMsg:  "failed to log IP attempt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCacheImpl.ExpectedCalls = nil
			tt.setupMock(mockCacheImpl)

			err := jwtCache.LogIPAttempt(ctx, tt.userID, tt.ipAddress)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockCacheImpl.AssertExpectations(t)
		})
	}
}

func TestJWTCache_GetIPAttempts(t *testing.T) {
	jwtCache, mockCacheImpl := SetupJWTCache(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		userID     string
		ipAddress  string
		setupMock  func(*mockCache)
		wantResult int64
		wantErr    bool
		errMsg     string
	}{
		{
			name:      "get existing attempts",
			userID:    "user123",
			ipAddress: "192.168.1.1",
			setupMock: func(m *mockCache) {
				expectedKey := fmt.Sprintf("%suser123:192.168.1.1", IPAttemptPrefix)
				m.On("Get", ctx, expectedKey).Return("5", nil)
			},
			wantResult: 5,
			wantErr:    false,
		},
		{
			name:      "key not found",
			userID:    "user456",
			ipAddress: "192.168.1.2",
			setupMock: func(m *mockCache) {
				expectedKey := fmt.Sprintf("%suser456:192.168.1.2", IPAttemptPrefix)
				m.On("Get", ctx, expectedKey).Return("", fmt.Errorf("key not found: %s", expectedKey))
			},
			wantResult: 0,
			wantErr:    false,
		},
		{
			name:      "cache error",
			userID:    "user789",
			ipAddress: "192.168.1.3",
			setupMock: func(m *mockCache) {
				expectedKey := fmt.Sprintf("%suser789:192.168.1.3", IPAttemptPrefix)
				m.On("Get", ctx, expectedKey).Return("", fmt.Errorf("some other error"))
			},
			wantResult: 0,
			wantErr:    true,
			errMsg:     "failed to get IP attempts",
		},
		{
			name:      "invalid number format",
			userID:    "user999",
			ipAddress: "192.168.1.4",
			setupMock: func(m *mockCache) {
				expectedKey := fmt.Sprintf("%suser999:192.168.1.4", IPAttemptPrefix)
				m.On("Get", ctx, expectedKey).Return("invalid", nil)
			},
			wantResult: 0,
			wantErr:    true,
			errMsg:     "failed to parse IP attempts count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCacheImpl.ExpectedCalls = nil
			tt.setupMock(mockCacheImpl)

			result, err := jwtCache.GetIPAttempts(ctx, tt.userID, tt.ipAddress)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}

			mockCacheImpl.AssertExpectations(t)
		})
	}
}

func TestJWTCache_BlacklistUser(t *testing.T) {
	jwtCache, mockCacheImpl := SetupJWTCache(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    string
		duration  time.Duration
		setupMock func(*mockCache)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful blacklist",
			userID:   "user123",
			duration: time.Hour,
			setupMock: func(m *mockCache) {
				expectedKey := UserBlacklistPrefix + "user123"
				m.On("Set", ctx, expectedKey, "blacklisted", time.Hour).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "cache error",
			userID:   "user456",
			duration: time.Hour,
			setupMock: func(m *mockCache) {
				expectedKey := UserBlacklistPrefix + "user456"
				m.On("Set", ctx, expectedKey, "blacklisted", time.Hour).Return(fmt.Errorf("cache error"))
			},
			wantErr: true,
			errMsg:  "failed to blacklist user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCacheImpl.ExpectedCalls = nil
			tt.setupMock(mockCacheImpl)

			err := jwtCache.BlacklistUser(ctx, tt.userID, tt.duration)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockCacheImpl.AssertExpectations(t)
		})
	}
}

func TestJWTCache_IsUserBlacklisted(t *testing.T) {
	jwtCache, mockCacheImpl := SetupJWTCache(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		userID     string
		setupMock  func(*mockCache)
		wantResult bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:   "user is blacklisted",
			userID: "blacklisted_user",
			setupMock: func(m *mockCache) {
				expectedKey := UserBlacklistPrefix + "blacklisted_user"
				m.On("Exists", ctx, expectedKey).Return(true, nil)
			},
			wantResult: true,
			wantErr:    false,
		},
		{
			name:   "user is not blacklisted",
			userID: "clean_user",
			setupMock: func(m *mockCache) {
				expectedKey := UserBlacklistPrefix + "clean_user"
				m.On("Exists", ctx, expectedKey).Return(false, nil)
			},
			wantResult: false,
			wantErr:    false,
		},
		{
			name:   "cache error",
			userID: "error_user",
			setupMock: func(m *mockCache) {
				expectedKey := UserBlacklistPrefix + "error_user"
				m.On("Exists", ctx, expectedKey).Return(false, fmt.Errorf("cache error"))
			},
			wantResult: false,
			wantErr:    true,
			errMsg:     "failed to check user blacklist status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCacheImpl.ExpectedCalls = nil
			tt.setupMock(mockCacheImpl)

			result, err := jwtCache.IsUserBlacklisted(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}

			mockCacheImpl.AssertExpectations(t)
		})
	}
}

func TestNewJWTCache(t *testing.T) {
	mockCacheImpl := &mockCache{}
	mockLogger := &mockLogger{}

	jwtCache := NewJWTCache(mockCacheImpl, mockLogger)

	assert.NotNil(t, jwtCache)

	// Verify it implements the interface
	var _ JWTCache = jwtCache
}

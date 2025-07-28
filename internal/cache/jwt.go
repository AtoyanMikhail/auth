package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/AtoyanMikhail/auth/internal/logger"
)

type jwtCache struct {
	cache  Cache
	logger logger.Logger
}

// NewJWTCache creates a new JWT cache instance
func NewJWTCache(cache Cache, l logger.Logger) JWTCache {
	return &jwtCache{
		cache:  cache,
		logger: l,
	}
}

// BlacklistToken blacklists token until expiresAt.
func (j *jwtCache) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	key := TokenBlacklistPrefix + tokenID
	ttl := time.Until(expiresAt)

	// If token is already expired, don't add it to blacklist
	if ttl <= 0 {
		j.logger.Debug("Token already expired, not adding to blacklist",
			logger.String("token_id", tokenID))
		return nil
	}

	err := j.cache.Set(ctx, key, "blacklisted", ttl)
	if err != nil {
		j.logger.Error("Failed to blacklist token",
			logger.String("token_id", tokenID),
			logger.Error(err))
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	j.logger.Info("Token blacklisted",
		logger.String("token_id", tokenID),
		logger.String("ttl", ttl.String()))

	return nil
}

// IsTokenBlacklisted checks whether the token is blacklisted
func (j *jwtCache) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := TokenBlacklistPrefix + tokenID

	exists, err := j.cache.Exists(ctx, key)
	if err != nil {
		j.logger.Error("Failed to check token blacklist status",
			logger.String("token_id", tokenID),
			logger.Error(err))
		return false, fmt.Errorf("failed to check token blacklist status: %w", err)
	}

	return exists, nil
}

// LogIPAttempt caches attempt to log from specific IP
func (j *jwtCache) LogIPAttempt(ctx context.Context, userID, ipAddress string) error {
	key := fmt.Sprintf("%s%s:%s", IPAttemptPrefix, userID, ipAddress)
	ttl := 24 * time.Hour // Track attempts for the last 24 hours

	count, err := j.cache.IncrementWithTTL(ctx, key, ttl)
	if err != nil {
		j.logger.Error("Failed to log IP attempt",
			logger.String("user_id", userID),
			logger.String("ip", ipAddress),
			logger.Error(err))
		return fmt.Errorf("failed to log IP attempt: %w", err)
	}

	j.logger.Info("IP attempt logged",
		logger.String("user_id", userID),
		logger.String("ip", ipAddress),
		logger.Int("attempts", int(count)))

	return nil
}

// GetIPAttempts returns an amount of attempts to login attempts from sertain IP in a period of token's lifespan
func (j *jwtCache) GetIPAttempts(ctx context.Context, userID, ipAddress string) (int64, error) {
	key := fmt.Sprintf("%s%s:%s", IPAttemptPrefix, userID, ipAddress)

	val, err := j.cache.Get(ctx, key)
	if err != nil {
		// If key not found, return 0 attempts
		if err.Error() == fmt.Sprintf("key not found: %s", key) {
			return 0, nil
		}
		j.logger.Error("Failed to get IP attempts",
			logger.String("user_id", userID),
			logger.String("ip", ipAddress),
			logger.Error(err))
		return 0, fmt.Errorf("failed to get IP attempts: %w", err)
	}

	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		j.logger.Error("Failed to parse IP attempts count",
			logger.String("value", val),
			logger.Error(err))
		return 0, fmt.Errorf("failed to parse IP attempts count: %w", err)
	}

	return count, nil
}

// BlacklistUser blacklists user for a set duration
func (j *jwtCache) BlacklistUser(ctx context.Context, userID string, duration time.Duration) error {
	key := UserBlacklistPrefix + userID

	err := j.cache.Set(ctx, key, "blacklisted", duration)
	if err != nil {
		j.logger.Error("Failed to blacklist user",
			logger.String("user_id", userID),
			logger.Error(err))
		return fmt.Errorf("failed to blacklist user: %w", err)
	}

	j.logger.Info("User blacklisted",
		logger.String("user_id", userID),
		logger.String("duration", duration.String()))

	return nil
}

// IsUserBlacklisted checks whether the user is blacklisted
func (j *jwtCache) IsUserBlacklisted(ctx context.Context, userID string) (bool, error) {
	key := UserBlacklistPrefix + userID

	exists, err := j.cache.Exists(ctx, key)
	if err != nil {
		j.logger.Error("Failed to check user blacklist status",
			logger.String("user_id", userID),
			logger.Error(err))
		return false, fmt.Errorf("failed to check user blacklist status: %w", err)
	}

	return exists, nil
}

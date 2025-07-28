package models

import (
	"context"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	Close() error
	RunMigrations(migrationsFilePath string) error
	GetActiveByUserID(ctx context.Context, userID string) (*RefreshToken, error)
	GetByID(ctx context.Context, id int) (*RefreshToken, error)
	MarkAsUsed(ctx context.Context, tokenID int) error
	DeleteAllByUserID(ctx context.Context, userID string) error
	Delete(ctx context.Context, tokenID int) error
	CleanExpired(ctx context.Context) (int64, error)
	GetAllActiveByUserID(ctx context.Context, userID string) ([]*RefreshToken, error)
}

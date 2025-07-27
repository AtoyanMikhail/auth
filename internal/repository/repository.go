package repository

import (
	"context"

	"github.com/AtoyanMikhail/auth/internal/repository/models"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	Close() error 
	RunMigrations(migrationsFilePath string) error
	GetActiveByUserID(ctx context.Context, userID string) (*models.RefreshToken, error)
	GetByID(ctx context.Context, id int) (*models.RefreshToken, error)
	MarkAsUsed(ctx context.Context, tokenID int) error
	DeleteAllByUserID(ctx context.Context, userID string) error
	Delete(ctx context.Context, tokenID int) error
	CleanExpired(ctx context.Context) (int64, error)
	GetAllActiveByUserID(ctx context.Context, userID string) ([]*models.RefreshToken, error)
}

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AtoyanMikhail/auth/internal/config"
	"github.com/AtoyanMikhail/auth/internal/logger"
	"github.com/AtoyanMikhail/auth/internal/repository/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" //used for migrations
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //postgres driver
)

type refreshTokenRepo struct {
	db  *sqlx.DB
	l   logger.Logger
	cfg config.DatabaseConfig
}

func NewRefreshTokenRepository(cfg config.DatabaseConfig, l logger.Logger) (models.RefreshTokenRepository, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open db connection: %v", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("could not establish db connection: %v", err)
	}

	return &refreshTokenRepo{db: db, l: l, cfg: cfg}, nil
}

func (r *refreshTokenRepo) Close() error {
	return r.db.Close()
}

func (r *refreshTokenRepo) RunMigrations(migrationsPath string) error {
	driver, err := postgres.WithInstance(r.db.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (r *refreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip_address, expires_at)
		VALUES (:user_id, :token_hash, :user_agent, :ip_address, :expires_at)
		RETURNING id, created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		r.l.Error("Failed to prepare query", logger.Error(err))
		return fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	err = stmt.QueryRowxContext(ctx, token).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		r.l.Error("Failed to execute insert query", logger.Error(err))
		return err
	}

	r.l.Info("Refresh token created", logger.Int("id", token.ID), logger.String("user_id", token.UserID))
	return nil
}

func (r *refreshTokenRepo) GetActiveByUserID(ctx context.Context, userID string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address, created_at, expires_at, is_used, updated_at
		FROM refresh_tokens
		WHERE user_id = $1 AND expires_at > NOW() AND is_used = false
		ORDER BY created_at DESC
		LIMIT 1`

	token := &models.RefreshToken{}
	err := r.db.GetContext(ctx, token, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active refresh token found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

func (r *refreshTokenRepo) GetByID(ctx context.Context, id int) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address, created_at, expires_at, is_used, updated_at
		FROM refresh_tokens
		WHERE id = $1`

	token := &models.RefreshToken{}
	err := r.db.GetContext(ctx, token, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("refresh token with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

func (r *refreshTokenRepo) MarkAsUsed(ctx context.Context, tokenID int) error {
	query := `
		UPDATE refresh_tokens 
		SET is_used = true, updated_at = NOW() 
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		r.l.Error("Failed to mark token as used", logger.Error(err), logger.Int("token_id", tokenID))
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.l.Error("Failed to get rows affected after mark as used", logger.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.l.Warn("Token not found for mark as used", logger.Int("token_id", tokenID))
		return fmt.Errorf("token with id %d not found", tokenID)
	}

	r.l.Info("Refresh token marked as used", logger.Int("token_id", tokenID))
	return nil
}

func (r *refreshTokenRepo) DeleteAllByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete tokens for user %s: %w", userID, err)
	}

	return nil
}

func (r *refreshTokenRepo) Delete(ctx context.Context, tokenID int) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		r.l.Error("Failed to delete token", logger.Error(err), logger.Int("token_id", tokenID))
		return fmt.Errorf("failed to delete token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.l.Error("Failed to get rows affected after delete", logger.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.l.Warn("Token not found for delete", logger.Int("token_id", tokenID))
		return fmt.Errorf("token with id %d not found", tokenID)
	}

	r.l.Info("Refresh token deleted", logger.Int("token_id", tokenID))
	return nil
}

func (r *refreshTokenRepo) CleanExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to clean expired tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

func (r *refreshTokenRepo) GetAllActiveByUserID(ctx context.Context, userID string) ([]*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address, created_at, expires_at, is_used, updated_at
		FROM refresh_tokens
		WHERE user_id = $1 AND expires_at > NOW() AND is_used = false
		ORDER BY created_at DESC`

	var tokens []*models.RefreshToken
	err := r.db.SelectContext(ctx, &tokens, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tokens for user %s: %w", userID, err)
	}

	return tokens, nil
}

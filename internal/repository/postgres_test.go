package repository

import (
	"context"
	"database/sql"
	_ "database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/AtoyanMikhail/auth/internal/config"
	"github.com/AtoyanMikhail/auth/internal/logger"
	"github.com/AtoyanMikhail/auth/internal/repository/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock logger
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...logger.Field) {}
func (m *mockLogger) Info(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Warn(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Error(msg string, fields ...logger.Field) {}
func (m *mockLogger) Fatal(msg string, fields ...logger.Field) {}
func (m *mockLogger) Panic(msg string, fields ...logger.Field) {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                               { return nil }
func (m *mockLogger) SetLevel(level logger.Level)               {}

// Test repo initialization helper
func SetupTestRepo(t *testing.T) (*refreshTokenRepo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "postgres")
	
	repo := &refreshTokenRepo{
		db:  sqlxDB,
		l:   &mockLogger{},
		cfg: config.DatabaseConfig{},
	}

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

// Test token initialization helper
func createTestToken() *models.RefreshToken {
	return &models.RefreshToken{
		UserID:    "test-user-id",
		TokenHash: "test-hash",
		UserAgent: "test-agent",
		IPAddress: "192.168.1.1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsUsed:    false,
	}
}

func TestRefreshTokenRepo_Create(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	tests := []struct {
		name    string
		token   *models.RefreshToken
		mockFn  func(sqlmock.Sqlmock, *models.RefreshToken)
		wantErr bool
		errMsg  string
	}{
		{
			name:  "successful create",
			token: createTestToken(),
			mockFn: func(m sqlmock.Sqlmock, token *models.RefreshToken) {
				m.ExpectPrepare(`INSERT INTO refresh_tokens`).
					ExpectQuery().
					WithArgs(token.UserID, token.TokenHash, token.UserAgent, token.IPAddress, token.ExpiresAt).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(1, time.Now(), time.Now()))
			},
			wantErr: false,
		},
		{
			name:  "prepare statement error",
			token: createTestToken(),
			mockFn: func(m sqlmock.Sqlmock, token *models.RefreshToken) {
				m.ExpectPrepare(`INSERT INTO refresh_tokens`).
					WillReturnError(fmt.Errorf("prepare error"))
			},
			wantErr: true,
			errMsg:  "failed to prepare query",
		},
		{
			name:  "query execution error",
			token: createTestToken(),
			mockFn: func(m sqlmock.Sqlmock, token *models.RefreshToken) {
				m.ExpectPrepare(`INSERT INTO refresh_tokens`).
					ExpectQuery().
					WithArgs(token.UserID, token.TokenHash, token.UserAgent, token.IPAddress, token.ExpiresAt).
					WillReturnError(fmt.Errorf("query error"))
			},
			wantErr: true,
			errMsg:  "query error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock, tt.token)

			err := repo.Create(context.Background(), tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.token.ID)
				assert.NotZero(t, tt.token.CreatedAt)
				assert.NotZero(t, tt.token.UpdatedAt)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_GetActiveByUserID(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	userID := "test-user-id"
	expectedToken := createTestToken()
	expectedToken.ID = 1
	expectedToken.CreatedAt = time.Now()
	expectedToken.UpdatedAt = time.Now()

	tests := []struct {
		name    string
		userID  string
		mockFn  func(sqlmock.Sqlmock)
		want    *models.RefreshToken
		wantErr bool
		errMsg  string
	}{
		{
			name:   "successful get",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "token_hash", "user_agent", "ip_address", 
					"created_at", "expires_at", "is_used", "updated_at",
				}).AddRow(
					expectedToken.ID, expectedToken.UserID, expectedToken.TokenHash,
					expectedToken.UserAgent, expectedToken.IPAddress, expectedToken.CreatedAt,
					expectedToken.ExpiresAt, expectedToken.IsUsed, expectedToken.UpdatedAt,
				)
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			want:    expectedToken,
			wantErr: false,
		},
		{
			name:   "no rows found",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
			errMsg:  "no active refresh token found",
		},
		{
			name:   "database error",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnError(fmt.Errorf("database error"))
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed to get refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			result, err := repo.GetActiveByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ID, result.ID)
				assert.Equal(t, tt.want.UserID, result.UserID)
				assert.Equal(t, tt.want.TokenHash, result.TokenHash)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_GetByID(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	tokenID := 1
	expectedToken := createTestToken()
	expectedToken.ID = tokenID
	expectedToken.CreatedAt = time.Now()
	expectedToken.UpdatedAt = time.Now()

	tests := []struct {
		name    string
		tokenID int
		mockFn  func(sqlmock.Sqlmock)
		want    *models.RefreshToken
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful get",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "token_hash", "user_agent", "ip_address",
					"created_at", "expires_at", "is_used", "updated_at",
				}).AddRow(
					expectedToken.ID, expectedToken.UserID, expectedToken.TokenHash,
					expectedToken.UserAgent, expectedToken.IPAddress, expectedToken.CreatedAt,
					expectedToken.ExpiresAt, expectedToken.IsUsed, expectedToken.UpdatedAt,
				)
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE id = \$1`).
					WithArgs(tokenID).
					WillReturnRows(rows)
			},
			want:    expectedToken,
			wantErr: false,
		},
		{
			name:    "token not found",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE id = \$1`).
					WithArgs(tokenID).
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
			errMsg:  "refresh token with id 1 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			result, err := repo.GetByID(context.Background(), tt.tokenID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ID, result.ID)
				assert.Equal(t, tt.want.UserID, result.UserID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_MarkAsUsed(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	tokenID := 1

	tests := []struct {
		name    string
		tokenID int
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful mark as used",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`UPDATE refresh_tokens SET is_used = true`).
					WithArgs(tokenID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:    "token not found",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`UPDATE refresh_tokens SET is_used = true`).
					WithArgs(tokenID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errMsg:  "token with id 1 not found",
		},
		{
			name:    "database error",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`UPDATE refresh_tokens SET is_used = true`).
					WithArgs(tokenID).
					WillReturnError(fmt.Errorf("database error"))
			},
			wantErr: true,
			errMsg:  "failed to mark token as used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			err := repo.MarkAsUsed(context.Background(), tt.tokenID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_Delete(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	tokenID := 1

	tests := []struct {
		name    string
		tokenID int
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful delete",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE id = \$1`).
					WithArgs(tokenID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:    "token not found",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE id = \$1`).
					WithArgs(tokenID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errMsg:  "token with id 1 not found",
		},
		{
			name:    "database error",
			tokenID: tokenID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE id = \$1`).
					WithArgs(tokenID).
					WillReturnError(fmt.Errorf("database error"))
			},
			wantErr: true,
			errMsg:  "failed to delete token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			err := repo.Delete(context.Background(), tt.tokenID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_DeleteAllByUserID(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	userID := "test-user-id"

	tests := []struct {
		name    string
		userID  string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name:   "successful delete all",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnResult(sqlmock.NewResult(0, 3))
			},
			wantErr: false,
		},
		{
			name:   "database error",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnError(fmt.Errorf("database error"))
			},
			wantErr: true,
			errMsg:  "failed to delete tokens for user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			err := repo.DeleteAllByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_CleanExpired(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		want     int64
		wantErr  bool
		errMsg   string
	}{
		{
			name: "successful clean",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE expires_at < NOW\(\)`).
					WillReturnResult(sqlmock.NewResult(0, 5))
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "no expired tokens",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE expires_at < NOW\(\)`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM refresh_tokens WHERE expires_at < NOW\(\)`).
					WillReturnError(fmt.Errorf("database error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to clean expired tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			result, err := repo.CleanExpired(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, int64(0), result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_GetAllActiveByUserID(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	userID := "test-user-id"
	token1 := createTestToken()
	token1.ID = 1
	token1.CreatedAt = time.Now()
	token1.UpdatedAt = time.Now()

	token2 := createTestToken()
	token2.ID = 2
	token2.CreatedAt = time.Now().Add(-time.Hour)
	token2.UpdatedAt = time.Now().Add(-time.Hour)

	tests := []struct {
		name    string
		userID  string
		mockFn  func(sqlmock.Sqlmock)
		want    []*models.RefreshToken
		wantErr bool
		errMsg  string
	}{
		{
			name:   "successful get all",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "token_hash", "user_agent", "ip_address",
					"created_at", "expires_at", "is_used", "updated_at",
				}).
					AddRow(token1.ID, token1.UserID, token1.TokenHash, token1.UserAgent,
						token1.IPAddress, token1.CreatedAt, token1.ExpiresAt, token1.IsUsed, token1.UpdatedAt).
					AddRow(token2.ID, token2.UserID, token2.TokenHash, token2.UserAgent,
						token2.IPAddress, token2.CreatedAt, token2.ExpiresAt, token2.IsUsed, token2.UpdatedAt)

				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			want:    []*models.RefreshToken{token1, token2},
			wantErr: false,
		},
		{
			name:   "no tokens found",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "token_hash", "user_agent", "ip_address",
					"created_at", "expires_at", "is_used", "updated_at",
				})
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			want:    []*models.RefreshToken{},
			wantErr: false,
		},
		{
			name:   "database error",
			userID: userID,
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT .+ FROM refresh_tokens WHERE user_id = \$1`).
					WithArgs(userID).
					WillReturnError(fmt.Errorf("database error"))
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed to get active tokens for user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(mock)

			result, err := repo.GetAllActiveByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.want), len(result))
				for i, expectedToken := range tt.want {
					assert.Equal(t, expectedToken.ID, result[i].ID)
					assert.Equal(t, expectedToken.UserID, result[i].UserID)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepo_Close(t *testing.T) {
	repo, mock, cleanup := SetupTestRepo(t)
	defer cleanup()

	mock.ExpectClose()

	err := repo.Close()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
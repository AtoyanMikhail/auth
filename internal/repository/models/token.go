package models

import "time"

type RefreshToken struct {
	ID        int       `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	TokenHash string    `db:"token_hash" json:"token_hash"`
	UserAgent string    `db:"user_agent" json:"user_agent"`
	IPAddress string    `db:"ip_address" json:"ip_address"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	IsUsed    bool      `db:"is_used" json:"is_used"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

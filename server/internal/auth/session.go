package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserInfo is the minimal user data attached to every authenticated request.
type UserInfo struct {
	UserID      string
	DisplayName string
}

// CreateSession inserts a new 90-day session and returns its ID.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, expires_at)
		VALUES ($1::uuid, NOW() + INTERVAL '90 days')
		RETURNING id::text
	`, userID).Scan(&id)
	return id, err
}

// ValidateSession looks up a session by ID, checking expiry.
// Returns ErrNoRows (pgx) when the session doesn't exist or is expired.
func ValidateSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) (UserInfo, error) {
	var info UserInfo
	err := pool.QueryRow(ctx, `
		SELECT s.user_id::text, u.display_name
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1::uuid AND s.expires_at > NOW()
	`, sessionID).Scan(&info.UserID, &info.DisplayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserInfo{}, pgx.ErrNoRows
		}
		return UserInfo{}, err
	}
	return info, nil
}

// DeleteSession removes a session row.
func DeleteSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1::uuid`, sessionID)
	return err
}

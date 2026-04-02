package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mssola/useragent"
)

// UserInfo is the minimal user data attached to every authenticated request.
type UserInfo struct {
	UserID      string
	DisplayName string
}

// ParseUADisplay converts a raw User-Agent string into a short human-readable
// label such as "Safari 17 on iOS 17" or "Chrome 122 on macOS".
func ParseUADisplay(raw string) string {
	if raw == "" {
		return "Unknown"
	}
	ua := useragent.New(raw)

	browser, version := ua.Browser()
	os := ua.OS()

	// Trim patch version: "17.4.1" → "17.4"
	short := func(v string) string {
		parts := splitN(v, ".", 3)
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1]
		}
		return v
	}

	if browser != "" && os != "" {
		return fmt.Sprintf("%s %s on %s", browser, short(version), os)
	}
	if browser != "" {
		return fmt.Sprintf("%s %s", browser, short(version))
	}
	if os != "" {
		return os
	}
	// Fall back to first 80 chars of raw UA
	if len(raw) > 80 {
		return raw[:80]
	}
	return raw
}

func splitN(s, sep string, n int) []string {
	var parts []string
	for i := 0; i < n-1; i++ {
		idx := indexByte(s, sep[0])
		if idx < 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	return append(parts, s)
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// CreateSession inserts a new 90-day session and returns its ID.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID, userAgentRaw, ipAddress string) (string, error) {
	uaDisplay := ParseUADisplay(userAgentRaw)
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO session (user_id, expires_at, user_agent, ua_display, ip_address)
		VALUES ($1::uuid, NOW() + INTERVAL '90 days', $2, $3, $4)
		RETURNING id::text
	`, userID, userAgentRaw, uaDisplay, ipAddress).Scan(&id)
	return id, err
}

// ValidateSession looks up a session by ID, checking expiry.
// Returns ErrNoRows (pgx) when the session doesn't exist or is expired.
func ValidateSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) (UserInfo, error) {
	var info UserInfo
	err := pool.QueryRow(ctx, `
		SELECT s.user_id::text, u.display_name
		FROM session s
		JOIN name u ON u.id = s.user_id
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
	_, err := pool.Exec(ctx, `DELETE FROM session WHERE id = $1::uuid`, sessionID)
	return err
}

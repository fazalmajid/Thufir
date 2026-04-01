package middleware

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"thufir/internal/auth"
)

type contextKey string

const userKey contextKey = "user"

// RequireAuth is a chi middleware that validates the session cookie.
// On success it stores UserInfo in the request context.
// On failure it writes 401 and stops the chain.
func RequireAuth(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			info, err := auth.ValidateSession(r.Context(), pool, cookie.Value)
			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromCtx retrieves UserInfo stored by RequireAuth.
func UserFromCtx(ctx context.Context) (auth.UserInfo, bool) {
	v, ok := ctx.Value(userKey).(auth.UserInfo)
	return v, ok
}


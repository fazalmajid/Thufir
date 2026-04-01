package sync

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	mw "thufir/internal/middleware"
)

type quickAddRequest struct {
	Title string  `json:"title"`
	Notes *string `json:"notes"`
}

// HandleQuickAdd creates a single inbox task from a simple {title, notes} payload.
// Intended for use by browser bookmarklets.
func HandleQuickAdd(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := mw.UserFromCtx(r.Context())
		if !ok {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req quickAddRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
			http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
			return
		}

		id := uuid.New().String()
		now := time.Now().UTC().Format(time.RFC3339)

		conn, err := pool.Acquire(r.Context())
		if err != nil {
			http.Error(w, `{"error":"db"}`, http.StatusInternalServerError)
			return
		}
		defer conn.Release()

		_, err = conn.Exec(r.Context(), `
			INSERT INTO task (
				id, user_id, title, notes, status, is_completed, is_flagged,
				priority, tags, sort_order, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3, $4,
				'inbox', false, false, 0, '{}', 0, $5::timestamptz, $5::timestamptz
			)`,
			id, user.UserID, req.Title, req.Notes, now,
		)
		if err != nil {
			http.Error(w, `{"error":"insert failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"` + id + `"}`)) //nolint:errcheck
	}
}

// Package sync implements the RxDB checkpoint-based replication protocol.
//
// Each collection (tasks, projects, areas) exposes two HTTP endpoints:
//
//	POST /api/rxdb/{collection}/pull  – client fetches changes since a checkpoint
//	POST /api/rxdb/{collection}/push  – client pushes local changes to the server
//
// Checkpoint format: {"updated_at":"<RFC3339>","id":"<UUID>"}
// A null checkpoint means "start from the beginning".
//
// Pull response:  {"documents":[…],"checkpoint":{…}|null}
// Push body:      array of {newDocumentState, assumedMasterState|null}
// Push response:  [] (no conflicts) or [conflicting master docs]
package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"thufir/internal/middleware"
)

// ── shared types ──────────────────────────────────────────────────────────────

// Checkpoint is the opaque cursor the client echoes back on each pull.
type Checkpoint struct {
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
}

// pullRequest is the JSON body the client sends to the pull endpoint.
type pullRequest struct {
	Checkpoint *json.RawMessage `json:"checkpoint"` // null on first pull
	Limit      int              `json:"limit"`
}

// pullResponse is the JSON the server returns.
type pullResponse struct {
	Documents  []json.RawMessage `json:"documents"`
	Checkpoint *Checkpoint       `json:"checkpoint"` // null when no documents
}

// pushRow is one element of the client's push payload.
type pushRow struct {
	NewDocumentState   json.RawMessage  `json:"newDocumentState"`
	AssumedMasterState *json.RawMessage `json:"assumedMasterState"` // nil means "new doc"
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeCheckpoint parses a raw JSON checkpoint (possibly null).
// Returns zero Checkpoint and false when null/absent.
func decodeCheckpoint(raw *json.RawMessage) (Checkpoint, bool) {
	if raw == nil {
		return Checkpoint{}, false
	}
	var cp Checkpoint
	if err := json.Unmarshal(*raw, &cp); err != nil {
		return Checkpoint{}, false
	}
	return cp, true
}

// fetchCurrentDoc returns the current DB row as a JSON document (with _deleted),
// or nil if the row does not belong to this user / does not exist.
func fetchCurrentDoc(ctx context.Context, pool *pgxpool.Pool, table, id, userID string) (json.RawMessage, error) {
	q := `SELECT (to_jsonb(t) - 'user_id' || jsonb_build_object('_deleted', t.deleted_at IS NOT NULL))::text
	      FROM ` + table + ` t
	      WHERE t.id = $1::uuid AND t.user_id = $2::uuid`
	var raw string
	err := pool.QueryRow(ctx, q, id, userID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// userIDFromRequest extracts the authenticated user ID via the middleware context.
func userIDFromRequest(r *http.Request) (string, bool) {
	info, ok := middleware.UserFromCtx(r.Context())
	return info.UserID, ok
}

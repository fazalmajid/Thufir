package sync

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HandlePull returns the pull handler for the given table name.
// The table must have columns: id, user_id, updated_at, deleted_at (all others
// are returned via to_jsonb()).
func HandlePull(table string, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromRequest(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req pullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}
		if req.Limit <= 0 || req.Limit > 500 {
			req.Limit = 100
		}

		cp, hasCP := decodeCheckpoint(req.Checkpoint)

		// Build query — two variants depending on whether we have a checkpoint.
		var docRows []json.RawMessage
		var lastUpdatedAt time.Time
		var lastID string

		var queryErr error
		if !hasCP {
			// First sync: return everything ordered by (updated_at, id).
			rows, err := pool.Query(r.Context(), `
				SELECT
					(to_jsonb(t) - 'user_id' || jsonb_build_object('_deleted', t.deleted_at IS NOT NULL))::text,
					t.updated_at,
					t.id::text
				FROM `+table+` t
				WHERE t.user_id = $1::uuid
				ORDER BY t.updated_at ASC, t.id ASC
				LIMIT $2
			`, userID, req.Limit)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "db error")
				return
			}
			defer rows.Close()
			for rows.Next() {
				var raw string
				if err := rows.Scan(&raw, &lastUpdatedAt, &lastID); err != nil {
					queryErr = err
					break
				}
				docRows = append(docRows, json.RawMessage(raw))
			}
			queryErr = rows.Err()
		} else {
			// Subsequent sync: return only rows newer than the checkpoint.
			rows, err := pool.Query(r.Context(), `
				SELECT
					(to_jsonb(t) - 'user_id' || jsonb_build_object('_deleted', t.deleted_at IS NOT NULL))::text,
					t.updated_at,
					t.id::text
				FROM `+table+` t
				WHERE t.user_id = $1::uuid
				  AND (
					t.updated_at > $2::timestamptz
					OR (t.updated_at = $2::timestamptz AND t.id::text > $3)
				  )
				ORDER BY t.updated_at ASC, t.id ASC
				LIMIT $4
			`, userID, cp.UpdatedAt, cp.ID, req.Limit)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "db error")
				return
			}
			defer rows.Close()
			for rows.Next() {
				var raw string
				if err := rows.Scan(&raw, &lastUpdatedAt, &lastID); err != nil {
					queryErr = err
					break
				}
				docRows = append(docRows, json.RawMessage(raw))
			}
			queryErr = rows.Err()
		}

		if queryErr != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}

		resp := pullResponse{Documents: docRows}
		if len(docRows) > 0 {
			resp.Checkpoint = &Checkpoint{UpdatedAt: lastUpdatedAt.UTC(), ID: lastID}
		} else if hasCP {
			resp.Checkpoint = &cp // echo back the same checkpoint when nothing new
		}
		if resp.Documents == nil {
			resp.Documents = []json.RawMessage{} // never return null
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

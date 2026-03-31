package sync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HandlePush returns the push handler for the given collection.
// collection must be one of "task", "project", or "area".
func HandlePush(collection string, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromRequest(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var rows []pushRow
		if err := json.NewDecoder(r.Body).Decode(&rows); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		var conflicts []json.RawMessage

		for _, row := range rows {
			// Extract the document ID so we can look up the current server state.
			var docID struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(row.NewDocumentState, &docID); err != nil || docID.ID == "" {
				writeErr(w, http.StatusBadRequest, "document missing id")
				return
			}

			// Fetch current server state (with lock).
			var currentRaw string
			var currentUpdatedAt time.Time
			err := tx.QueryRow(r.Context(), `
				SELECT
					(to_jsonb(t) - 'user_id' || jsonb_build_object('_deleted', t.deleted_at IS NOT NULL))::text,
					t.updated_at
				FROM `+collection+` t
				WHERE t.id = $1::uuid AND t.user_id = $2::uuid
				FOR UPDATE
			`, docID.ID, userID).Scan(&currentRaw, &currentUpdatedAt)

			rowExists := true
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					writeErr(w, http.StatusInternalServerError, "db error")
					return
				}
				rowExists = false
			}

			// Conflict detection.
			if rowExists && row.AssumedMasterState != nil {
				var assumed struct {
					UpdatedAt string `json:"updated_at"`
				}
				if err := json.Unmarshal(*row.AssumedMasterState, &assumed); err == nil && assumed.UpdatedAt != "" {
					assumedTime, _ := time.Parse(time.RFC3339Nano, assumed.UpdatedAt)
					if !assumedTime.Equal(currentUpdatedAt.UTC()) {
						conflicts = append(conflicts, json.RawMessage(currentRaw))
						continue // skip this write
					}
				}
			} else if rowExists && row.AssumedMasterState == nil {
				// Client thinks it's new but server already has it → conflict.
				conflicts = append(conflicts, json.RawMessage(currentRaw))
				continue
			}

			// Apply write.
			if err := upsertDocument(r.Context(), tx, collection, userID, row.NewDocumentState); err != nil { //nolint:contextcheck
				writeErr(w, http.StatusInternalServerError, "upsert failed: "+err.Error())
				return
			}
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeErr(w, http.StatusInternalServerError, "commit")
			return
		}

		if conflicts == nil {
			conflicts = []json.RawMessage{}
		}
		writeJSON(w, http.StatusOK, conflicts)
	}
}

// upsertDocument routes to the collection-specific upsert function.
func upsertDocument(ctx context.Context, tx pgx.Tx, collection, userID string, doc json.RawMessage) error {
	switch collection {
	case "task":
		return upsertTask(ctx, tx, userID, doc)
	case "project":
		return upsertProject(ctx, tx, userID, doc)
	case "area":
		return upsertArea(ctx, tx, userID, doc)
	}
	return errors.New("unknown collection: " + collection)
}

package sync

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// taskDoc mirrors the task JSON document shape sent by RxDB.
type taskDoc struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Notes         *string  `json:"notes"`
	ProjectID     *string  `json:"project_id"`
	AreaID        *string  `json:"area_id"`
	ParentTaskID  *string  `json:"parent_task_id"`
	Status        string   `json:"status"`
	IsCompleted   bool     `json:"is_completed"`
	CompletedAt   *string  `json:"completed_at"`
	StartDate     *string  `json:"start_date"`
	Deadline      *string  `json:"deadline"`
	ScheduledDate *string  `json:"scheduled_date"`
	StartTime     *string  `json:"start_time"`
	ReminderTime  *string  `json:"reminder_time"`
	IsFlagged     bool     `json:"is_flagged"`
	Priority      int      `json:"priority"`
	Tags          []string `json:"tags"`
	SortOrder     int      `json:"sort_order"`
	CreatedAt     string   `json:"created_at"`
	DeletedAt     *string  `json:"deleted_at"`
}

func upsertTask(ctx context.Context, tx pgx.Tx, userID string, raw json.RawMessage) error {
	var d taskDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}

	// The updated_at is always set to NOW() — server clock is authoritative.
	// deleted_at is client-controlled (soft delete / restore); pass it through as-is.
	_, err := tx.Exec(ctx, `
		INSERT INTO task (
			id, user_id, title, notes, project_id, area_id, parent_task_id,
			status, is_completed, completed_at, start_date, deadline,
			scheduled_date, start_time, reminder_time, is_flagged, priority,
			tags, sort_order, created_at, updated_at, deleted_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5::uuid, $6::uuid, $7::uuid,
			$8, $9, $10::timestamptz, $11::date, $12::date,
			$13::date, $14::time, $15::timestamptz, $16, $17,
			$18, $19, $20::timestamptz, NOW(), $21::timestamptz
		)
		ON CONFLICT (id) DO UPDATE SET
			title          = EXCLUDED.title,
			notes          = EXCLUDED.notes,
			project_id     = EXCLUDED.project_id,
			area_id        = EXCLUDED.area_id,
			parent_task_id = EXCLUDED.parent_task_id,
			status         = EXCLUDED.status,
			is_completed   = EXCLUDED.is_completed,
			completed_at   = EXCLUDED.completed_at,
			start_date     = EXCLUDED.start_date,
			deadline       = EXCLUDED.deadline,
			scheduled_date = EXCLUDED.scheduled_date,
			start_time     = EXCLUDED.start_time,
			reminder_time  = EXCLUDED.reminder_time,
			is_flagged     = EXCLUDED.is_flagged,
			priority       = EXCLUDED.priority,
			tags           = EXCLUDED.tags,
			sort_order     = EXCLUDED.sort_order,
			updated_at     = NOW(),
			deleted_at     = EXCLUDED.deleted_at
		WHERE task.user_id = $2::uuid
	`,
		d.ID, userID, d.Title, d.Notes, d.ProjectID, d.AreaID, d.ParentTaskID,
		d.Status, d.IsCompleted, d.CompletedAt, d.StartDate, d.Deadline,
		d.ScheduledDate, d.StartTime, d.ReminderTime, d.IsFlagged, d.Priority,
		d.Tags, d.SortOrder, d.CreatedAt, d.DeletedAt,
	)
	return err
}

// projectDoc mirrors the project JSON document shape.
type projectDoc struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Notes       *string  `json:"notes"`
	AreaID      *string  `json:"area_id"`
	Status      string   `json:"status"`
	Deadline    *string  `json:"deadline"`
	Tags        []string `json:"tags"`
	SortOrder   int      `json:"sort_order"`
	CreatedAt   string   `json:"created_at"`
	CompletedAt *string  `json:"completed_at"`
	DeletedAt   *string  `json:"deleted_at"`
}

func upsertProject(ctx context.Context, tx pgx.Tx, userID string, raw json.RawMessage) error {
	var d projectDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}
	if d.Status == "" {
		d.Status = "active"
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO project (
			id, user_id, name, notes, area_id, status, deadline,
			tags, sort_order, created_at, updated_at, completed_at, deleted_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5::uuid, $6, $7::date,
			$8, $9, $10::timestamptz, NOW(), $11::timestamptz, $12::timestamptz
		)
		ON CONFLICT (id) DO UPDATE SET
			name         = EXCLUDED.name,
			notes        = EXCLUDED.notes,
			area_id      = EXCLUDED.area_id,
			status       = EXCLUDED.status,
			deadline     = EXCLUDED.deadline,
			tags         = EXCLUDED.tags,
			sort_order   = EXCLUDED.sort_order,
			completed_at = EXCLUDED.completed_at,
			updated_at   = NOW(),
			deleted_at   = EXCLUDED.deleted_at
		WHERE project.user_id = $2::uuid
	`,
		d.ID, userID, d.Name, d.Notes, d.AreaID, d.Status, d.Deadline,
		d.Tags, d.SortOrder, d.CreatedAt, d.CompletedAt, d.DeletedAt,
	)
	return err
}

// areaDoc mirrors the area JSON document shape.
type areaDoc struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Color     *string `json:"color"`
	Icon      *string `json:"icon"`
	SortOrder int     `json:"sort_order"`
	CreatedAt string  `json:"created_at"`
	DeletedAt *string `json:"deleted_at"`
}

func upsertArea(ctx context.Context, tx pgx.Tx, userID string, raw json.RawMessage) error {
	var d areaDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO area (
			id, user_id, name, color, icon, sort_order, created_at, updated_at, deleted_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7::timestamptz, NOW(), $8::timestamptz
		)
		ON CONFLICT (id) DO UPDATE SET
			name       = EXCLUDED.name,
			color      = EXCLUDED.color,
			icon       = EXCLUDED.icon,
			sort_order = EXCLUDED.sort_order,
			updated_at = NOW(),
			deleted_at = EXCLUDED.deleted_at
		WHERE area.user_id = $2::uuid
	`,
		d.ID, userID, d.Name, d.Color, d.Icon, d.SortOrder, d.CreatedAt, d.DeletedAt,
	)
	return err
}

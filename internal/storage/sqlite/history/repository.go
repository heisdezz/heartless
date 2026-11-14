package history

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domain "heartless/internal/domain/history"
)

var ErrNotFound = errors.New("history entry not found")

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, entry *domain.Entry) error {
	if entry == nil {
		return errors.New("history entry is nil")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO execution_history (id, workspace_id, request_id, method, url, status_code, duration_ms, error, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	workspace_id = excluded.workspace_id,
	request_id = excluded.request_id,
	method = excluded.method,
	url = excluded.url,
	status_code = excluded.status_code,
	duration_ms = excluded.duration_ms,
	error = excluded.error
`, entry.ID, entry.WorkspaceID, entry.RequestID, entry.Method, entry.URL, entry.StatusCode, entry.DurationMS, entry.Error, formatTime(entry.CreatedAt))
	return err
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Entry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, request_id, method, url, status_code, duration_ms, error, created_at FROM execution_history WHERE id = ?`, id)
	return scan(row)
}

func (r *Repository) List(ctx context.Context, workspaceID string, limit int) ([]*domain.Entry, error) {
	q := `SELECT id, workspace_id, request_id, method, url, status_code, duration_ms, error, created_at FROM execution_history`
	args := []any{}
	if workspaceID != "" {
		q += ` WHERE workspace_id = ?`
		args = append(args, workspaceID)
	}
	q += ` ORDER BY created_at DESC, id DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Entry
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM execution_history WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Clear(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		_, err := r.db.ExecContext(ctx, `DELETE FROM execution_history`)
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM execution_history WHERE workspace_id = ?`, workspaceID)
	return err
}

// Prune keeps the newest limit entries for a workspace and deletes older ones.
func (r *Repository) Prune(ctx context.Context, workspaceID string, limit int) error {
	if limit <= 0 {
		return r.Clear(ctx, workspaceID)
	}
	if workspaceID == "" {
		_, err := r.db.ExecContext(ctx, `
DELETE FROM execution_history
WHERE id NOT IN (
	SELECT id FROM execution_history ORDER BY created_at DESC, id DESC LIMIT ?
)`, limit)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
DELETE FROM execution_history
WHERE workspace_id = ? AND id NOT IN (
	SELECT id FROM execution_history WHERE workspace_id = ? ORDER BY created_at DESC, id DESC LIMIT ?
)`, workspaceID, workspaceID, limit)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(s scanner) (*domain.Entry, error) {
	var e domain.Entry
	var createdAt string
	if err := s.Scan(&e.ID, &e.WorkspaceID, &e.RequestID, &e.Method, &e.URL, &e.StatusCode, &e.DurationMS, &e.Error, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	return &e, nil
}

const timeLayout = "2006-01-02T15:04:05Z"

func formatTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

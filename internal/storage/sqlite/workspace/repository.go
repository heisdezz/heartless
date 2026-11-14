package workspace

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domain "heartless/internal/domain/workspace"
)

// Repository provides SQLite persistence for workspaces.
type Repository struct {
	db *sql.DB
}

// New creates a workspace repository backed by SQLite.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Workspace, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, created_at, updated_at FROM workspaces WHERE id = ?`, id)
	return scanWorkspace(row)
}

func (r *Repository) Save(ctx context.Context, ws *domain.Workspace) error {
	if ws == nil {
		return errors.New("workspace is nil")
	}
	if ws.CreatedAt.IsZero() {
		ws.CreatedAt = time.Now().UTC()
	}
	ws.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
INSERT INTO workspaces (id, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	description = excluded.description,
	updated_at = excluded.updated_at
`, ws.ID, ws.Name, ws.Description, formatTime(ws.CreatedAt), formatTime(ws.UpdatedAt))
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, _ string) ([]*domain.Workspace, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at FROM workspaces ORDER BY updated_at DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Workspace
	for rows.Next() {
		ws, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, rows.Err()
}

func (r *Repository) Exists(ctx context.Context, name string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT 1 FROM workspaces WHERE name = ? LIMIT 1`, name)
	var one int
	err := row.Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(s scanner) (*domain.Workspace, error) {
	var ws domain.Workspace
	var createdAt, updatedAt string
	if err := s.Scan(&ws.ID, &ws.Name, &ws.Description, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	ws.CreatedAt = parseTime(createdAt)
	ws.UpdatedAt = parseTime(updatedAt)
	return &ws, nil
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

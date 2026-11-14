package environment

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domain "heartless/internal/domain/environment"
)

// Repository provides SQLite persistence for environments.
type Repository struct {
	db *sql.DB
}

// New creates an environment repository backed by SQLite.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Environment, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, description, created_at, updated_at FROM environments WHERE id = ?`, id)
	return scanEnvironment(row)
}

func (r *Repository) Save(ctx context.Context, env *domain.Environment, workspaceID string) error {
	if env == nil {
		return errors.New("environment is nil")
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = time.Now().UTC()
	}
	if workspaceID != "" {
		env.WorkspaceID = workspaceID
	}
	env.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
INSERT INTO environments (id, workspace_id, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	workspace_id = excluded.workspace_id,
	name = excluded.name,
	description = excluded.description,
	updated_at = excluded.updated_at
`, env.ID, env.WorkspaceID, env.Name, env.Description, formatTime(env.CreatedAt), formatTime(env.UpdatedAt))
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM environments WHERE id = ?`, id)
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

func (r *Repository) List(ctx context.Context, workspaceID string) ([]*domain.Environment, error) {
	query := `SELECT id, workspace_id, name, description, created_at, updated_at FROM environments`
	args := []any{}
	if workspaceID != "" {
		query += ` WHERE workspace_id = ?`
		args = append(args, workspaceID)
	}
	query += ` ORDER BY updated_at DESC, name ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Environment
	for rows.Next() {
		env, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEnvironment(s scanner) (*domain.Environment, error) {
	var env domain.Environment
	var createdAt, updatedAt string
	if err := s.Scan(&env.ID, &env.WorkspaceID, &env.Name, &env.Description, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	env.CreatedAt = parseTime(createdAt)
	env.UpdatedAt = parseTime(updatedAt)
	return &env, nil
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

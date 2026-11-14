package variable

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domain "heartless/internal/domain/variable"
)

// Repository provides SQLite persistence for variables.
type Repository struct {
	db *sql.DB
}

// New creates a variable repository backed by SQLite.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Variable, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, environment_id, name, value, type, description, secret, scope, created_at, updated_at FROM variables WHERE id = ?`, id)
	return scanVariable(row)
}

func (r *Repository) Save(ctx context.Context, id, workspaceID, environmentID string, v *domain.Variable) error {
	if v == nil {
		return errors.New("variable is nil")
	}
	if id == "" {
		id = v.ID
	}
	if id == "" {
		return errors.New("variable id is required")
	}
	if workspaceID != "" {
		v.WorkspaceID = workspaceID
	}
	if environmentID != "" {
		v.EnvironmentID = environmentID
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	v.UpdatedAt = time.Now().UTC()
	v.ID = id

	secret := 0
	if v.Secret {
		secret = 1
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO variables (id, workspace_id, environment_id, name, value, type, description, secret, scope, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	workspace_id = excluded.workspace_id,
	environment_id = excluded.environment_id,
	name = excluded.name,
	value = excluded.value,
	type = excluded.type,
	description = excluded.description,
	secret = excluded.secret,
	scope = excluded.scope,
	updated_at = excluded.updated_at
`, id, v.WorkspaceID, v.EnvironmentID, v.Name, v.Value, v.Type, v.Description, secret, v.Scope, formatTime(v.CreatedAt), formatTime(v.UpdatedAt))
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM variables WHERE id = ?`, id)
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

func (r *Repository) List(ctx context.Context, workspaceID, environmentID string) ([]*domain.Variable, error) {
	query := `SELECT id, workspace_id, environment_id, name, value, type, description, secret, scope, created_at, updated_at FROM variables WHERE 1=1`
	args := []any{}
	if workspaceID != "" {
		query += ` AND workspace_id = ?`
		args = append(args, workspaceID)
	}
	if environmentID != "" {
		query += ` AND environment_id = ?`
		args = append(args, environmentID)
	}
	query += ` ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Variable
	for rows.Next() {
		v, err := scanVariable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanVariable(s scanner) (*domain.Variable, error) {
	var v domain.Variable
	var secret int
	var createdAt, updatedAt string
	if err := s.Scan(&v.ID, &v.WorkspaceID, &v.EnvironmentID, &v.Name, &v.Value, &v.Type, &v.Description, &secret, &v.Scope, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	v.Secret = secret != 0
	v.CreatedAt = parseTime(createdAt)
	v.UpdatedAt = parseTime(updatedAt)
	return &v, nil
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

package workspace

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	domainreq "heartless/internal/domain/request"
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

// MigrateDB is a legacy no-op; schema changes are applied by sqlite migrations.
func MigrateDB(db *sql.DB) error {
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Get retrieves a workspace by ID.
func (r *Repository) Get(ctx context.Context, id string) (*domain.Workspace, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, created_at, updated_at, default_headers, default_auth, default_tls_insecure, auto_inherit_auth FROM workspaces WHERE id = ?`, id)
	return scanWorkspace(row)
}

// Save persists a workspace.
func (r *Repository) Save(ctx context.Context, ws *domain.Workspace) error {
	if ws == nil {
		return errors.New("workspace is nil")
	}
	if ws.CreatedAt.IsZero() {
		ws.CreatedAt = time.Now().UTC()
	}
	ws.UpdatedAt = time.Now().UTC()

	// Marshal the new fields to JSON strings for storage
	if ws.DefaultHeaders == nil {
		ws.DefaultHeaders = map[string]string{}
	}
	defaultHeadersJSON, _ := json.Marshal(ws.DefaultHeaders)
	defaultAuthJSON, _ := json.Marshal(ws.DefaultAuth)

	_, err := r.db.ExecContext(ctx, `
INSERT INTO workspaces (id, name, description, created_at, updated_at, default_headers, default_auth, default_tls_insecure, auto_inherit_auth)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	description = excluded.description,
	updated_at = excluded.updated_at,
	default_headers = excluded.default_headers,
	default_auth = excluded.default_auth,
	default_tls_insecure = excluded.default_tls_insecure,
	auto_inherit_auth = excluded.auto_inherit_auth
`, ws.ID, ws.Name, ws.Description, formatTime(ws.CreatedAt), formatTime(ws.UpdatedAt), string(defaultHeadersJSON), string(defaultAuthJSON), boolToInt(ws.DefaultTLSInsecure), boolToInt(ws.AutoInheritAuth))
	return err
}

// Delete removes a workspace by ID.
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

// List returns all workspaces.
func (r *Repository) List(ctx context.Context, _ string) ([]*domain.Workspace, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at, default_headers, default_auth, default_tls_insecure, auto_inherit_auth FROM workspaces ORDER BY updated_at DESC, name ASC`)
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

// GetActiveEnvironmentID returns the persisted active environment ID for a workspace.
func (r *Repository) GetActiveEnvironmentID(ctx context.Context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", nil
	}
	var envID string
	err := r.db.QueryRowContext(ctx, `SELECT active_environment_id FROM workspace_state WHERE workspace_id = ?`, workspaceID).Scan(&envID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return envID, nil
}

// SetActiveEnvironmentID persists the active environment ID for a workspace.
func (r *Repository) SetActiveEnvironmentID(ctx context.Context, workspaceID, environmentID string) error {
	if workspaceID == "" {
		return errors.New("workspace id is required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO workspace_state (workspace_id, active_environment_id, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	active_environment_id = excluded.active_environment_id,
	updated_at = excluded.updated_at
`, workspaceID, environmentID, formatTime(time.Now().UTC()))
	return err
}

// Exists checks if a workspace with the given name already exists.
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
	var createdAt, updatedAt, defaultHeadersJSON, defaultAuthJSON string
	var defaultTLSInsecure, autoInheritAuth int
	if err := s.Scan(&ws.ID, &ws.Name, &ws.Description, &createdAt, &updatedAt, &defaultHeadersJSON, &defaultAuthJSON, &defaultTLSInsecure, &autoInheritAuth); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	ws.CreatedAt = parseTime(createdAt)
	ws.UpdatedAt = parseTime(updatedAt)
	// Decode default_headers JSON
	if err := json.Unmarshal([]byte(defaultHeadersJSON), &ws.DefaultHeaders); err != nil {
		ws.DefaultHeaders = map[string]string{}
	}
	// Decode defaultAuth JSON
	if err := json.Unmarshal([]byte(defaultAuthJSON), &ws.DefaultAuth); err != nil {
		ws.DefaultAuth = domainreq.Auth{}
	}
	ws.DefaultTLSInsecure = defaultTLSInsecure != 0
	ws.AutoInheritAuth = autoInheritAuth != 0
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

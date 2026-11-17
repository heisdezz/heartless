package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"heartless/internal/config"
)

// Repository persists the single row of application settings in SQLite.
type Repository struct {
	db *sql.DB
}

// New creates a settings repository backed by SQLite.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Load returns the stored settings, or the defaults when none are saved yet.
func (r *Repository) Load(ctx context.Context) (config.ApplicationSettings, error) {
	var themeJSON, interfaceJSON, pluginsJSON string
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT theme, interface, plugins, created_at, updated_at FROM app_settings WHERE id = 1`,
	).Scan(&themeJSON, &interfaceJSON, &pluginsJSON, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return config.DefaultApplicationSettings(), nil
	}
	if err != nil {
		return config.ApplicationSettings{}, err
	}

	s := config.DefaultApplicationSettings()
	if err := json.Unmarshal([]byte(themeJSON), &s.Theme); err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(interfaceJSON), &s.Interface); err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(pluginsJSON), &s.Plugins); err != nil {
		return s, err
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return s, nil
}

// GetActiveWorkspaceID returns the workspace selected on the previous launch.
func (r *Repository) GetActiveWorkspaceID(ctx context.Context) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT active_workspace_id FROM app_settings WHERE id = 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

// SetActiveWorkspaceID records the selected workspace for the next launch. It
// leaves the rest of the settings row alone, so saving the theme or switching
// workspaces never overwrites the other's value.
func (r *Repository) SetActiveWorkspaceID(ctx context.Context, id string) error {
	themeJSON, interfaceJSON, pluginsJSON, err := settingsJSON(config.DefaultApplicationSettings())
	if err != nil {
		return err
	}
	now := formatTime(time.Now().UTC())
	_, err = r.db.ExecContext(ctx, `
INSERT INTO app_settings (id, theme, interface, plugins, active_workspace_id, created_at, updated_at)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	active_workspace_id = excluded.active_workspace_id,
	updated_at = excluded.updated_at
`, themeJSON, interfaceJSON, pluginsJSON, id, now, now)
	return err
}

func settingsJSON(s config.ApplicationSettings) (theme, iface, plugins string, err error) {
	themeJSON, err := json.Marshal(s.Theme)
	if err != nil {
		return "", "", "", err
	}
	interfaceJSON, err := json.Marshal(s.Interface)
	if err != nil {
		return "", "", "", err
	}
	pluginsJSON, err := json.Marshal(s.Plugins)
	if err != nil {
		return "", "", "", err
	}
	return string(themeJSON), string(interfaceJSON), string(pluginsJSON), nil
}

// Save writes the settings, replacing the stored row.
func (r *Repository) Save(ctx context.Context, s config.ApplicationSettings) error {
	themeJSON, interfaceJSON, pluginsJSON, err := settingsJSON(s)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	_, err = r.db.ExecContext(ctx, `
INSERT INTO app_settings (id, theme, interface, plugins, created_at, updated_at)
VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	theme = excluded.theme,
	interface = excluded.interface,
	plugins = excluded.plugins,
	updated_at = excluded.updated_at
`, themeJSON, interfaceJSON, pluginsJSON, formatTime(s.CreatedAt), formatTime(s.UpdatedAt))
	return err
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

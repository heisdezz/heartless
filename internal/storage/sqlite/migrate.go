package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

type migration struct {
	version int
	name    string
	up      string
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_schema",
		up: `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspaces (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS environments (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_environments_workspace_id ON environments(workspace_id);

CREATE TABLE IF NOT EXISTS variables (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL DEFAULT '',
	environment_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	value TEXT NOT NULL DEFAULT '',
	type TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	secret INTEGER NOT NULL DEFAULT 0,
	scope TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_variables_workspace_id ON variables(workspace_id);
CREATE INDEX IF NOT EXISTS idx_variables_environment_id ON variables(environment_id);
`,
	},
	{
		version: 2,
		name:    "workspace_active_environment",
		up: `
CREATE TABLE IF NOT EXISTS workspace_state (
	workspace_id TEXT PRIMARY KEY,
	active_environment_id TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL
);
`,
	},
	{
		version: 3,
		name:    "collections",
		up: `
CREATE TABLE IF NOT EXISTS collections (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_collections_workspace_id ON collections(workspace_id);

CREATE TABLE IF NOT EXISTS folders (
	id TEXT PRIMARY KEY,
	collection_id TEXT NOT NULL DEFAULT '',
	parent_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_folders_collection_id ON folders(collection_id);
CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id);

CREATE TABLE IF NOT EXISTS requests (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL DEFAULT '',
	collection_id TEXT NOT NULL DEFAULT '',
	folder_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL DEFAULT '',
	method TEXT NOT NULL DEFAULT '',
	url TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '',
	body_type TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_requests_workspace_id ON requests(workspace_id);
CREATE INDEX IF NOT EXISTS idx_requests_collection_id ON requests(collection_id);
CREATE INDEX IF NOT EXISTS idx_requests_folder_id ON requests(folder_id);
`,
	},
	{
		version: 4,
		name:    "execution_history",
		up: `
CREATE TABLE IF NOT EXISTS execution_history (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL DEFAULT '',
	request_id TEXT NOT NULL DEFAULT '',
	method TEXT NOT NULL DEFAULT '',
	url TEXT NOT NULL DEFAULT '',
	status_code INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	error TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_execution_history_workspace_created ON execution_history(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_execution_history_request_id ON execution_history(request_id);
`,
	},
	{
		version: 5,
		name:    "request_headers_params",
		up: `
ALTER TABLE requests ADD COLUMN headers TEXT NOT NULL DEFAULT '{}';
ALTER TABLE requests ADD COLUMN params TEXT NOT NULL DEFAULT '{}';
`,
	},
	{
		version: 6,
		name:    "history_response_payload",
		up: `
ALTER TABLE execution_history ADD COLUMN status_text TEXT NOT NULL DEFAULT '';
ALTER TABLE execution_history ADD COLUMN response_headers TEXT NOT NULL DEFAULT '{}';
ALTER TABLE execution_history ADD COLUMN body TEXT NOT NULL DEFAULT '';
ALTER TABLE execution_history ADD COLUMN content_type TEXT NOT NULL DEFAULT '';
ALTER TABLE execution_history ADD COLUMN size INTEGER NOT NULL DEFAULT 0;
`,
	},
	{
		version: 7,
		name:    "request_auth",
		up: `
ALTER TABLE requests ADD COLUMN auth TEXT NOT NULL DEFAULT '{}';
`,
	},
	{
		version: 8,
		name:    "workspace_defaults",
		up: `
ALTER TABLE workspaces ADD COLUMN default_headers TEXT NOT NULL DEFAULT '{}';
ALTER TABLE workspaces ADD COLUMN default_auth TEXT NOT NULL DEFAULT '{}';
ALTER TABLE workspaces ADD COLUMN default_tls_insecure INTEGER NOT NULL DEFAULT 0;
`,
	},
	{
		version: 9,
		name:    "app_settings",
		up: `
CREATE TABLE IF NOT EXISTS app_settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	theme TEXT NOT NULL DEFAULT '{}',
	interface TEXT NOT NULL DEFAULT '{}',
	plugins TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`,
	},
	{
		version: 10,
		name:    "history_response_metadata",
		up: `
ALTER TABLE execution_history ADD COLUMN response_id TEXT NOT NULL DEFAULT '';
ALTER TABLE execution_history ADD COLUMN response_request_id TEXT NOT NULL DEFAULT '';
ALTER TABLE execution_history ADD COLUMN response_duration_ns INTEGER NOT NULL DEFAULT 0;
ALTER TABLE execution_history ADD COLUMN effective_url TEXT NOT NULL DEFAULT '';
`,
	},
	{
		version: 11,
		name:    "request_script",
		up: `
ALTER TABLE requests ADD COLUMN script TEXT NOT NULL DEFAULT '';
`,
	},
	{
		version: 12,
		name:    "active_workspace",
		up: `
ALTER TABLE app_settings ADD COLUMN active_workspace_id TEXT NOT NULL DEFAULT '';
`,
	},
	{
		version: 13,
		name:    "workspace_auto_inherit_auth",
		up: `
ALTER TABLE workspaces ADD COLUMN auto_inherit_auth INTEGER NOT NULL DEFAULT 0;
`,
	},
	{
		version: 14,
		name:    "request_form",
		up: `
ALTER TABLE requests ADD COLUMN form TEXT NOT NULL DEFAULT '{}';
`,
	},
}

// Migrate applies pending schema migrations in order.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	sorted := append([]migration(nil), migrations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].version < sorted[j].version })

	for _, m := range sorted {
		if applied[m.version] {
			continue
		}
		if err := Tx(ctx, db, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, m.up); err != nil {
				return fmt.Errorf("apply migration %d (%s): %w", m.version, m.name, err)
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`,
				m.version, m.name, NowUTC().Format(timeLayout),
			); err != nil {
				return fmt.Errorf("record migration %d: %w", m.version, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

const timeLayout = "2006-01-02T15:04:05Z"

func appliedVersions(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("list schema_migrations: %w", err)
	}
	defer rows.Close()

	out := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		out[v] = true
	}
	return out, rows.Err()
}

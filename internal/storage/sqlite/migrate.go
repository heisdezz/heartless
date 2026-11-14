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

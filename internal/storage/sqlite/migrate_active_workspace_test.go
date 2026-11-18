package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	domainws "heartless/internal/domain/workspace"
	sqlitestore "heartless/internal/storage/sqlite"
	settingsrepo "heartless/internal/storage/sqlite/settings"
	workspacerepo "heartless/internal/storage/sqlite/workspace"
)

// TestMigrateAddsActiveWorkspaceToExistingDB checks the upgrade path: a
// database migrated before the active-workspace column existed gains it.
func TestMigrateAddsActiveWorkspaceToExistingDB(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := legacy.ExecContext(ctx, `
CREATE TABLE schema_migrations (
	version INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TEXT NOT NULL
);
CREATE TABLE app_settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	theme TEXT NOT NULL DEFAULT '{}',
	interface TEXT NOT NULL DEFAULT '{}',
	plugins TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE workspaces (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	default_headers TEXT NOT NULL DEFAULT '{}',
	default_auth TEXT NOT NULL DEFAULT '{}',
	default_tls_insecure INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE requests (
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
	updated_at TEXT NOT NULL,
	headers TEXT NOT NULL DEFAULT '{}',
	params TEXT NOT NULL DEFAULT '{}',
	auth TEXT NOT NULL DEFAULT '{}',
	script TEXT NOT NULL DEFAULT ''
);
`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	for v := 1; v <= 11; v++ {
		if _, err := legacy.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			v, fmt.Sprintf("legacy_%d", v), "2026-01-01T00:00:00Z",
		); err != nil {
			t.Fatalf("mark migration %d applied: %v", v, err)
		}
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	db, err := sqlitestore.Open(ctx, sqlitestore.Config{Path: path})
	if err != nil {
		t.Fatalf("open and migrate: %v", err)
	}
	defer db.Close()

	repo := settingsrepo.New(db)
	if err := repo.SetActiveWorkspaceID(ctx, "ws-legacy"); err != nil {
		t.Fatalf("set active workspace after migration: %v", err)
	}
	got, err := repo.GetActiveWorkspaceID(ctx)
	if err != nil || got != "ws-legacy" {
		t.Fatalf("active workspace after migration = %q, %v", got, err)
	}

	// The upgrade also gives workspaces the auto-inherit column, defaulted off.
	ws := domainws.NewWorkspace("Legacy")
	ws.ID = "ws-1"
	ws.AutoInheritAuth = true
	if err := workspacerepo.New(db).Save(ctx, ws); err != nil {
		t.Fatalf("save workspace after migration: %v", err)
	}
	saved, err := workspacerepo.New(db).Get(ctx, ws.ID)
	if err != nil {
		t.Fatalf("get workspace after migration: %v", err)
	}
	if !saved.AutoInheritAuth {
		t.Fatal("auto_inherit_auth did not round-trip after migration")
	}
}

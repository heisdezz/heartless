package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	sqlitestore "heartless/internal/storage/sqlite"
	settingsrepo "heartless/internal/storage/sqlite/settings"
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
}

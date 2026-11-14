package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"heartless/internal/domain/environment"
	"heartless/internal/domain/workspace"
	sqlitestore "heartless/internal/storage/sqlite"
	envrepo "heartless/internal/storage/sqlite/environment"
	wsrepo "heartless/internal/storage/sqlite/workspace"
)

func TestOpenMigrateAndRepositories(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "heartless.db")
	db, err := sqlitestore.Open(context.Background(), sqlitestore.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	wsr := wsrepo.New(db)
	ws := workspace.NewWorkspace("Main")
	ws.ID = "ws-1"
	if err := wsr.Save(context.Background(), ws); err != nil {
		t.Fatalf("save workspace: %v", err)
	}
	gotWS, err := wsr.Get(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if gotWS.Name != "Main" {
		t.Fatalf("workspace name = %q", gotWS.Name)
	}
	list, err := wsr.List(context.Background(), "")
	if err != nil || len(list) != 1 {
		t.Fatalf("list workspaces: %v len=%d", err, len(list))
	}
	exists, err := wsr.Exists(context.Background(), "Main")
	if err != nil || !exists {
		t.Fatalf("exists: %v %v", exists, err)
	}

	er := envrepo.New(db)
	env := environment.NewEnvironment("Dev")
	env.ID = "env-1"
	if err := er.Save(context.Background(), env, "ws-1"); err != nil {
		t.Fatalf("save env: %v", err)
	}
	gotEnv, err := er.Get(context.Background(), "env-1")
	if err != nil || gotEnv.Name != "Dev" {
		t.Fatalf("get env: %v %+v", err, gotEnv)
	}
	envs, err := er.List(context.Background(), "ws-1")
	if err != nil || len(envs) != 1 {
		t.Fatalf("list envs: %v len=%d", err, len(envs))
	}

	if _, err := db.Exec(`PRAGMA foreign_keys`); err != nil {
		t.Fatalf("pragma: %v", err)
	}
}

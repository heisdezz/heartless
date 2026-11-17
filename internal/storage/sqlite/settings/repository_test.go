package settings_test

import (
	"context"
	"path/filepath"
	"testing"

	"heartless/internal/config"
	sqlitestore "heartless/internal/storage/sqlite"
	settingsrepo "heartless/internal/storage/sqlite/settings"
)

func TestActiveWorkspaceSurvivesSettingsSave(t *testing.T) {
	ctx := context.Background()
	db, err := sqlitestore.Open(ctx, sqlitestore.Config{Path: filepath.Join(t.TempDir(), "heartless.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	repo := settingsrepo.New(db)
	got, err := repo.GetActiveWorkspaceID(ctx)
	if err != nil || got != "" {
		t.Fatalf("initial active workspace = %q, %v", got, err)
	}

	if err := repo.SetActiveWorkspaceID(ctx, "ws-2"); err != nil {
		t.Fatalf("set active workspace: %v", err)
	}
	s := config.DefaultApplicationSettings()
	s.Theme.Name = "nord"
	if err := repo.Save(ctx, s); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	got, err = repo.GetActiveWorkspaceID(ctx)
	if err != nil || got != "ws-2" {
		t.Fatalf("active workspace after settings save = %q, %v", got, err)
	}
	loaded, err := repo.Load(ctx)
	if err != nil || loaded.Theme.Name != "nord" {
		t.Fatalf("settings after active workspace change = %+v, %v", loaded, err)
	}

	if err := repo.SetActiveWorkspaceID(ctx, "ws-1"); err != nil {
		t.Fatalf("change active workspace: %v", err)
	}
	got, err = repo.GetActiveWorkspaceID(ctx)
	if err != nil || got != "ws-1" {
		t.Fatalf("changed active workspace = %q, %v", got, err)
	}
}

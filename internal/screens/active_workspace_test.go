package screens

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"heartless/internal/app"
	domainws "heartless/internal/domain/workspace"
)

// TestActiveWorkspaceRememberedAcrossRelaunch switches away from the workspace
// chosen on first launch, then checks a fresh shell opens on that choice.
func TestActiveWorkspaceRememberedAcrossRelaunch(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "heartless.db")
	app.SetDefaultDBPath(dbPath)

	open := func() *app.Services {
		t.Helper()
		services, err := app.Open(ctx)
		if err != nil {
			t.Fatalf("open services: %v", err)
		}
		return services
	}

	first := open()
	workspaces := make([]*domainws.Workspace, 0, 2)
	for _, name := range []string{"Cren", "Exitit"} {
		ws := domainws.NewWorkspace(name)
		if err := first.Workspaces.Save(ctx, ws); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
		workspaces = append(workspaces, ws)
	}

	st := shellState{router: ui.NewRouter("/")}
	st.load(ctx, first)
	other := workspaces[0].ID
	if st.workspaceID == other {
		other = workspaces[1].ID
	}
	otherName := workspaces[0].Name
	if other == workspaces[1].ID {
		otherName = workspaces[1].Name
	}
	st.selectWorkspace(other, first)
	first.DB.Close()

	second := open()
	defer second.DB.Close()
	reopened := shellState{router: ui.NewRouter("/")}
	reopened.load(ctx, second)
	if reopened.workspaceID != other {
		t.Fatalf("active workspace after relaunch = %q, want %q", reopened.workspaceID, other)
	}
	tt := ui.NewTester(func(c *ui.Context) { App(c, second) }, 1000, 700)
	if !tt.HasText(otherName) {
		t.Fatalf("title bar after relaunch has no %q, texts %q", otherName, tt.Texts())
	}
}

// TestStaleActiveWorkspaceFallsBack checks a saved ID for a deleted workspace
// is replaced with an existing one and that the replacement is saved.
func TestStaleActiveWorkspaceFallsBack(t *testing.T) {
	ctx := context.Background()
	services := newTestServices(t)
	if err := services.Settings.SetActiveWorkspaceID(ctx, "ws-gone"); err != nil {
		t.Fatalf("set stale active workspace: %v", err)
	}

	st := shellState{router: ui.NewRouter("/")}
	st.load(ctx, services)
	if st.workspaceID != "ws-1" {
		t.Fatalf("active workspace = %q, want ws-1", st.workspaceID)
	}
	got, err := services.Settings.GetActiveWorkspaceID(ctx)
	if err != nil || got != "ws-1" {
		t.Fatalf("persisted active workspace = %q, %v", got, err)
	}
}

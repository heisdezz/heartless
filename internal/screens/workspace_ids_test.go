package screens

import (
	"context"
	"path/filepath"
	"testing"

	"heartless/internal/app"
	domainws "heartless/internal/domain/workspace"
)

// TestWorkspacesAccumulateAcrossRelaunch creates workspaces in two separate
// app instances pointing at the same database, as relaunching the app does, and
// checks each one is kept instead of overwriting the previous ones.
func TestWorkspacesAccumulateAcrossRelaunch(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "heartless.db")

	open := func() *app.Services {
		app.SetDefaultDBPath(dbPath)
		svc, err := app.Open(ctx)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		return svc
	}

	// First launch: create two workspaces through the editor paths.
	first := open()
	for _, name := range []string{"Cren", "Exitit"} {
		ws := domainws.NewWorkspace(name)
		if err := first.Workspaces.Save(ctx, ws); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}
	first.DB.Close()

	// Second launch, same database: the counter would have reset before.
	second := open()
	defer second.DB.Close()
	for _, name := range []string{"Greenmouse", "Yaak"} {
		ws := domainws.NewWorkspace(name)
		if err := second.Workspaces.Save(ctx, ws); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	list, err := second.Workspaces.List(ctx, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 4 {
		names := make([]string, 0, len(list))
		for _, ws := range list {
			names = append(names, ws.Name)
		}
		t.Fatalf("workspaces = %d (%v), want 4", len(list), names)
	}
	seen := map[string]bool{}
	for _, ws := range list {
		if seen[ws.ID] {
			t.Fatalf("duplicate id %q", ws.ID)
		}
		seen[ws.ID] = true
	}
}

// TestWorkspaceIDsAreUnique checks two workspaces made in one session differ.
func TestWorkspaceIDsAreUnique(t *testing.T) {
	a := domainws.NewWorkspace("A")
	b := domainws.NewWorkspace("B")
	if a.ID == b.ID {
		t.Fatalf("ids equal: %q", a.ID)
	}
}

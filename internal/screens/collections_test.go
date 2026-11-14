package screens

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
)

// TestCollectionTree seeds a nested tree and checks the sidebar renders it and
// that context-menu actions create children.
func TestCollectionTree(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	col := domaincol.New("ws-1", "api")
	if err := services.Collections.SaveCollection(ctx, col); err != nil {
		t.Fatalf("save collection: %v", err)
	}
	folder := domaincol.NewFolder(col.ID, "", "login")
	if err := services.Collections.SaveFolder(ctx, folder); err != nil {
		t.Fatalf("save folder: %v", err)
	}
	req := domainreq.New("ws-1", "Log in", domainreq.KindHTTP)
	req.CollectionID = col.ID
	req.FolderID = folder.ID
	req.Method = "POST"
	if err := services.Collections.SaveRequest(ctx, req); err != nil {
		t.Fatalf("save request: %v", err)
	}

	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/")}
	st.load(ctx, services)
	st.setExpanded(col.ID, true)
	st.setExpanded(folder.ID, true)

	if len(st.collections) != 1 || len(st.childFolders(col.ID, "")) != 1 || len(st.childRequests(col.ID, folder.ID)) != 1 {
		t.Fatalf("tree = %d collections, folder children %d, requests %d",
			len(st.collections), len(st.childFolders(col.ID, "")), len(st.childRequests(col.ID, folder.ID)))
	}

	// The sidebar shows collection, folder and request names.
	tt := ui.NewTester(func(c *ui.Context) { App(c, services) }, 1000, 700)
	for _, want := range []string{"api", "login", "Log in"} {
		if !tt.HasText(want) {
			t.Fatalf("sidebar missing %q, texts %q", want, tt.Texts())
		}
		if r, ok := tt.Find(want); !ok || r.W == 0 || r.H == 0 {
			t.Fatalf("sidebar row %q has no size: %+v", want, r)
		}
	}
}

// TestFolderMenuActions checks that folder menu choices create the right child.
func TestFolderMenuActions(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	col := domaincol.New("ws-1", "api")
	if err := services.Collections.SaveCollection(ctx, col); err != nil {
		t.Fatalf("save collection: %v", err)
	}

	st := shellState{workspaceID: "ws-1"}
	st.reloadTree(ctx, services)

	st.applyFolderMenu(services, col.ID, "", menuFolder)
	st.reloadTree(ctx, services)
	folders := st.childFolders(col.ID, "")
	if len(folders) != 1 || folders[0].Name != "Folder 1" {
		t.Fatalf("folders = %+v", folders)
	}

	st.applyFolderMenu(services, col.ID, folders[0].ID, menuGraphQL)
	st.reloadTree(ctx, services)
	reqs := st.childRequests(col.ID, folders[0].ID)
	if len(reqs) != 1 || reqs[0].Kind != domainreq.KindGraphQL {
		t.Fatalf("requests = %+v", reqs)
	}

	st.applyFolderMenu(services, col.ID, folders[0].ID, menuDelete)
	st.reloadTree(ctx, services)
	if len(st.childFolders(col.ID, "")) != 0 || len(st.childRequests(col.ID, folders[0].ID)) != 0 {
		t.Fatalf("folder not deleted")
	}
}

// TestCollectionRepoRoundTrip exercises the new SQLite repository.
func TestCollectionRepoRoundTrip(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	col := domaincol.New("ws-1", "api")
	if err := services.Collections.SaveCollection(ctx, col); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := services.Collections.GetCollection(ctx, col.ID)
	if err != nil || got.Name != "api" {
		t.Fatalf("get = %+v err %v", got, err)
	}
	list, err := services.Collections.ListCollections(ctx, "ws-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list len %d err %v", len(list), err)
	}
	if err := services.Collections.DeleteCollection(ctx, col.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := services.Collections.GetCollection(ctx, col.ID); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

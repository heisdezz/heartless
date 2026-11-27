package screens

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const testPostmanV1 = `{
  "info": {"name": "API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "item": [
    {"name": "Users", "item": [
      {"name": "List", "request": {"method": "GET", "url": "https://example.com/v1/users"}},
      {"name": "RemoveMe", "request": {"method": "DELETE", "url": "https://example.com/v1/remove"}}
    ]}
  ]
}`

const testPostmanV2 = `{
  "info": {"name": "API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "item": [
    {"name": "Users", "item": [
      {"name": "List", "request": {"method": "GET", "url": "https://example.com/v2/users"}},
      {"name": "Create", "request": {"method": "POST", "url": "https://example.com/v2/users"}}
    ]}
  ]
}`

func TestImportNewCollectionImportsDirectly(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/")}
	st.load(context.Background(), services)

	st.openImportCollection()
	st.importCollectionData(services, []byte(testPostmanV1))

	if st.showImportConflict {
		t.Fatal("unexpected conflict modal on brand-new collection")
	}
	if st.showImportCollection {
		t.Fatal("import modal should close after successful import")
	}
	if len(st.collections) != 1 || st.collections[0].Name != "API" {
		t.Fatalf("collections = %+v", st.collections)
	}
}

func TestImportExistingCollectionConflictModalMerge(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/")}
	st.load(ctx, services)

	// Initial import.
	st.importCollectionData(services, []byte(testPostmanV1))
	if len(st.requests) != 2 {
		t.Fatalf("initial requests = %d, want 2", len(st.requests))
	}

	var listID, removeID string
	for i := range st.requests {
		r := st.requests[i]
		switch r.Name {
		case "List":
			listID = r.ID
			r.Script = `env.set("token", "abc");`
			if err := services.Collections.SaveRequest(ctx, &r); err != nil {
				t.Fatalf("save script: %v", err)
			}
			st.requests[i] = r
		case "RemoveMe":
			removeID = r.ID
			r.Script = `console.log("removed");`
			if err := services.Collections.SaveRequest(ctx, &r); err != nil {
				t.Fatalf("save script: %v", err)
			}
			st.requests[i] = r
		}
	}

	// Open List in editor state and select RemoveMe to verify state sync on merge.
	listReq, _ := st.requestByID(listID)
	_ = st.requestStateFor(listReq)
	removeReq, _ := st.requestByID(removeID)
	_ = st.requestStateFor(removeReq)
	st.selectedRequestID = removeID

	// Importing a collection with the same name pops up the conflict modal.
	st.openImportCollection()
	st.importCollectionData(services, []byte(testPostmanV2))
	if !st.showImportConflict {
		t.Fatalf("expected conflict modal to open, err=%q", st.importCollectionErr)
	}
	if st.showImportCollection {
		t.Fatal("expected import modal to close when conflict modal opens")
	}

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		importConflictModal(c, &st, services)
	}, 900, 700)
	tt.Frame()
	tt.Frame()

	for _, want := range []string{"Collection already exists", "Merge", "Add copy", "Cancel"} {
		if !tt.HasText(want) {
			t.Fatalf("conflict modal missing %q, texts %q", want, tt.Texts())
		}
	}

	if err := tt.Click("Merge"); err != nil {
		t.Fatalf("click Merge: %v", err)
	}
	tt.Frame()

	if st.showImportConflict {
		t.Fatalf("conflict modal still open after Merge, err=%q", st.importConflictErr)
	}
	if len(st.collections) != 1 || st.collections[0].Name != "API" {
		t.Fatalf("collections = %+v, want single merged API collection", st.collections)
	}

	gotList, err := services.Collections.GetRequest(ctx, listID)
	if err != nil {
		t.Fatalf("get kept request: %v", err)
	}
	if gotList.URL != "https://example.com/v2/users" {
		t.Fatalf("List URL = %q, want v2 URL", gotList.URL)
	}
	if gotList.Script != `env.set("token", "abc");` {
		t.Fatalf("List script broken after merge: %q", gotList.Script)
	}
	if rs := st.requestStates[listID]; rs == nil || rs.URL != "https://example.com/v2/users" || rs.Script != `env.set("token", "abc");` {
		t.Fatalf("open requestState not synced after merge: %+v", rs)
	}

	if _, err := services.Collections.GetRequest(ctx, removeID); err == nil {
		t.Fatal("expected RemoveMe request to be deleted after merge")
	}
	if st.selectedRequestID != "" {
		t.Fatalf("selectedRequestID = %q, want cleared after removed request deleted", st.selectedRequestID)
	}
	if _, ok := st.requestStates[removeID]; ok {
		t.Fatal("expected removed request state to be cleaned up")
	}
}

func TestImportExistingCollectionConflictModalAddCopy(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/")}
	st.load(ctx, services)

	st.importCollectionData(services, []byte(testPostmanV1))
	for i := range st.requests {
		if st.requests[i].Name == "List" {
			st.requests[i].Script = `env.set("keep", "1");`
			if err := services.Collections.SaveRequest(ctx, &st.requests[i]); err != nil {
				t.Fatalf("save script: %v", err)
			}
			break
		}
	}

	st.importCollectionData(services, []byte(testPostmanV2))
	if !st.showImportConflict {
		t.Fatal("expected conflict modal")
	}

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		importConflictModal(c, &st, services)
	}, 900, 700)
	tt.Frame()
	tt.Frame()

	if err := tt.Click("Add copy"); err != nil {
		t.Fatalf("click Add copy: %v", err)
	}
	tt.Frame()

	if st.showImportConflict {
		t.Fatal("conflict modal should close after Add copy")
	}
	if len(st.collections) != 2 {
		t.Fatalf("collections = %+v, want 2", st.collections)
	}
	names := map[string]bool{}
	for _, col := range st.collections {
		names[col.Name] = true
	}
	if !names["API"] || !names["API (imported)"] {
		t.Fatalf("collection names = %v", names)
	}

	// Original collection's request and script remain untouched.
	var origColID string
	for _, col := range st.collections {
		if col.Name == "API" {
			origColID = col.ID
		}
	}
	origReqs, err := services.Collections.ListRequests(ctx, "ws-1", origColID, "")
	if err != nil {
		t.Fatalf("list original requests: %v", err)
	}
	var found bool
	for _, r := range origReqs {
		if r.Name == "List" {
			found = true
			if r.URL != "https://example.com/v1/users" || r.Script != `env.set("keep", "1");` {
				t.Fatalf("original request modified: %+v", r)
			}
		}
	}
	if !found {
		t.Fatal("original List request missing")
	}
}

func TestImportExistingCollectionConflictModalCancel(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/")}
	st.load(ctx, services)

	st.importCollectionData(services, []byte(testPostmanV1))
	st.importCollectionData(services, []byte(testPostmanV2))
	if !st.showImportConflict {
		t.Fatal("expected conflict modal")
	}

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		importConflictModal(c, &st, services)
	}, 900, 700)
	tt.Frame()
	tt.Frame()

	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("click Cancel: %v", err)
	}
	tt.Frame()

	if st.showImportConflict {
		t.Fatal("conflict modal should close on Cancel")
	}
	if len(st.pendingImportData) != 0 {
		t.Fatal("pendingImportData should be cleared on Cancel")
	}
	if len(st.collections) != 1 || len(st.requests) != 2 {
		t.Fatalf("collections/requests modified on Cancel: %d/%d", len(st.collections), len(st.requests))
	}
}

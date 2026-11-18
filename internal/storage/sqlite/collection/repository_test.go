package collection_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"heartless/internal/domain/collection"
	"heartless/internal/domain/request"
	servicescol "heartless/internal/services/collection"
	sqlitestore "heartless/internal/storage/sqlite"
	colrepo "heartless/internal/storage/sqlite/collection"
)

// The repository must satisfy the application service contract.
var _ servicescol.Repository = (*colrepo.Repository)(nil)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlitestore.Open(context.Background(), sqlitestore.Config{
		Path: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRequestHeadersParamsRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := colrepo.New(openDB(t))

	req := request.New("ws-1", "List users", request.KindHTTP)
	req.Method = "GET"
	req.URL = "https://example.com/users"
	req.Headers = map[string]string{"Accept": "application/json", "X-Trace": "abc"}
	req.Params = map[string]string{"page": "1"}
	req.Form = map[string]string{"user": "ada"}

	if err := repo.SaveRequest(ctx, req); err != nil {
		t.Fatalf("save request: %v", err)
	}

	got, err := repo.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Headers["Accept"] != "application/json" || got.Headers["X-Trace"] != "abc" {
		t.Fatalf("headers = %v", got.Headers)
	}
	if got.Params["page"] != "1" {
		t.Fatalf("params = %v", got.Params)
	}
	if got.Form["user"] != "ada" {
		t.Fatalf("form = %v", got.Form)
	}

	list, err := repo.ListRequests(ctx, "ws-1", "", "")
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(list) != 1 || list[0].Headers["Accept"] != "application/json" || list[0].Form["user"] != "ada" {
		t.Fatalf("list = %+v", list)
	}

	// Empty maps must round-trip as empty maps, not nil.
	req.Headers = nil
	req.Params = nil
	req.Form = nil
	if err := repo.SaveRequest(ctx, req); err != nil {
		t.Fatalf("save request: %v", err)
	}
	got, err = repo.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if len(got.Headers) != 0 || len(got.Params) != 0 || len(got.Form) != 0 {
		t.Fatalf("headers/params/form = %v/%v/%v, want empty", got.Headers, got.Params, got.Form)
	}
}

func TestImportBatchCommits(t *testing.T) {
	ctx := context.Background()
	repo := colrepo.New(openDB(t))

	col := collection.New("ws-1", "API")
	parent := collection.NewFolder(col.ID, "", "Users")
	child := collection.NewFolder(col.ID, parent.ID, "Tokens")
	child.SortOrder = 1
	listReq := request.New("ws-1", "List", request.KindHTTP)
	listReq.CollectionID = col.ID
	listReq.FolderID = parent.ID
	listReq.Method = "GET"
	listReq.URL = "https://example.com/users"
	listReq.Headers = map[string]string{"Accept": "application/json"}

	if err := repo.ImportBatch(ctx,
		[]*collection.Collection{col},
		[]*collection.Folder{parent, child},
		[]*request.Request{listReq},
	); err != nil {
		t.Fatalf("import batch: %v", err)
	}

	cols, err := repo.ListCollections(ctx, "ws-1")
	if err != nil || len(cols) != 1 || cols[0].Name != "API" {
		t.Fatalf("collections = %+v err=%v", cols, err)
	}
	folders, err := repo.ListFolders(ctx, col.ID)
	if err != nil || len(folders) != 2 {
		t.Fatalf("folders = %+v err=%v", folders, err)
	}
	if folders[0].ParentID != "" || folders[1].ParentID != parent.ID {
		t.Fatalf("folder parents = %q/%q", folders[0].ParentID, folders[1].ParentID)
	}
	reqs, err := repo.ListRequests(ctx, "ws-1", col.ID, parent.ID)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("requests = %+v err=%v", reqs, err)
	}
	if reqs[0].Headers["Accept"] != "application/json" {
		t.Fatalf("request headers = %v", reqs[0].Headers)
	}
}

func TestImportBatchRollsBackOnFailure(t *testing.T) {
	ctx := context.Background()
	repo := colrepo.New(openDB(t))

	col := collection.New("ws-1", "API")
	// A nil request forces a failure after the collection insert.
	err := repo.ImportBatch(ctx,
		[]*collection.Collection{col},
		nil,
		[]*request.Request{nil},
	)
	if err == nil {
		t.Fatal("expected import batch to fail")
	}

	cols, listErr := repo.ListCollections(ctx, "ws-1")
	if listErr != nil {
		t.Fatalf("list collections: %v", listErr)
	}
	if len(cols) != 0 {
		t.Fatalf("rollback failed: collections = %+v", cols)
	}
}

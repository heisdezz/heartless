package importer_service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
	domainws "heartless/internal/domain/workspace"
	core "heartless/internal/importer"
	sqlitestore "heartless/internal/storage/sqlite"
	colrepo "heartless/internal/storage/sqlite/collection"
	wsrepo "heartless/internal/storage/sqlite/workspace"
)

const postmanDoc = `{
  "info": {"name": "API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "variable": [{"key": "baseUrl", "value": "https://x"}],
  "item": [
    {"name": "Users", "item": [
      {"name": "List", "request": {"method": "get", "header": [
        {"key": "Accept", "value": "application/json"},
        {"key": "X-Off", "value": "1", "disabled": true},
        {"key": "Accept", "value": "text/plain"}
      ], "url": "https://x/users"}}
    ]},
    {"name": "", "request": {"method": "", "url": "https://x/health"}}
  ]
}`

type fakeWorkspaces struct {
	ws  *domainws.Workspace
	err error
}

func (f fakeWorkspaces) Get(_ context.Context, _ string) (*domainws.Workspace, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.ws == nil {
		return nil, domainws.ErrNotFound
	}
	return f.ws, nil
}

type fakeStore struct {
	existing []*domaincol.Collection
	batchErr error
	cols     []*domaincol.Collection
	folders  []*domaincol.Folder
	reqs     []*domainreq.Request
}

func (f *fakeStore) ListCollections(_ context.Context, _ string) ([]*domaincol.Collection, error) {
	return f.existing, nil
}

func (f *fakeStore) ImportBatch(_ context.Context, cols []*domaincol.Collection, folders []*domaincol.Folder, reqs []*domainreq.Request) error {
	if f.batchErr != nil {
		return f.batchErr
	}
	f.cols = append(f.cols, cols...)
	f.folders = append(f.folders, folders...)
	f.reqs = append(f.reqs, reqs...)
	return nil
}

type stubImporter struct {
	draft *core.Draft
}

func (s stubImporter) Format() core.Format               { return "stub" }
func (s stubImporter) Detect([]byte) bool                { return true }
func (s stubImporter) Parse([]byte) (*core.Draft, error) { return s.draft, nil }

func newTestService(store *fakeStore) *Service {
	return New(Deps{
		Workspaces:  fakeWorkspaces{ws: domainws.NewWorkspace("Main")},
		Collections: store,
	})
}

func TestImportNormalizesAndPersists(t *testing.T) {
	store := &fakeStore{}
	svc := newTestService(store)

	res, err := svc.Import(context.Background(), "ws-1", []byte(postmanDoc), Options{})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.Format != core.FormatPostman {
		t.Fatalf("format = %q", res.Format)
	}
	if res.Collections != 1 || res.Folders != 1 || res.Requests != 2 {
		t.Fatalf("counts = %d/%d/%d, want 1/1/2", res.Collections, res.Folders, res.Requests)
	}
	if len(res.CreatedIDs) != 1 || len(store.cols) != 1 {
		t.Fatalf("created = %v, stored = %d", res.CreatedIDs, len(store.cols))
	}

	col := store.cols[0]
	if col.Name != "API" || col.WorkspaceID != "ws-1" {
		t.Fatalf("collection = %+v", col)
	}
	if len(store.folders) != 1 || store.folders[0].Name != "Users" || store.folders[0].CollectionID != col.ID {
		t.Fatalf("folders = %+v", store.folders)
	}
	folderID := store.folders[0].ID

	if len(store.reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(store.reqs))
	}
	list := store.reqs[0]
	if list.Name != "List" || list.Method != "GET" {
		t.Fatalf("list request = %+v", list)
	}
	if list.CollectionID != col.ID || list.FolderID != folderID {
		t.Fatalf("list linkage = %q/%q", list.CollectionID, list.FolderID)
	}
	// Disabled header dropped; duplicate name keeps the first value.
	if len(list.Headers) != 1 || list.Headers["Accept"] != "application/json" {
		t.Fatalf("headers = %v", list.Headers)
	}
	if list.SortOrder != 1 {
		t.Fatalf("list sort order = %d", list.SortOrder)
	}

	health := store.reqs[1]
	if health.Name != "Untitled Request" || health.Method != "GET" {
		t.Fatalf("health request = %+v", health)
	}
	if health.FolderID != "" || health.SortOrder != 2 {
		t.Fatalf("health linkage/sort = %q/%d", health.FolderID, health.SortOrder)
	}

	// Adapter issues (collection variables) and service warnings are reported.
	if !resultHasIssue(res, core.SeverityUnsupported, "variable") {
		t.Fatalf("missing adapter issue, issues = %+v", res.Issues)
	}
	if !resultHasIssue(res, core.SeverityWarning, "duplicate header") {
		t.Fatalf("missing duplicate header warning, issues = %+v", res.Issues)
	}
	if !resultHasIssue(res, core.SeverityWarning, "empty name") {
		t.Fatalf("missing empty name warning, issues = %+v", res.Issues)
	}
}

func TestImportConflictRename(t *testing.T) {
	store := &fakeStore{
		existing: []*domaincol.Collection{
			domaincol.New("ws-1", "API"),
			domaincol.New("ws-1", "API (imported)"),
		},
	}
	svc := newTestService(store)

	res, err := svc.Import(context.Background(), "ws-1", []byte(postmanDoc), Options{})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(store.cols) != 1 || store.cols[0].Name != "API (imported 2)" {
		t.Fatalf("collection names = %+v", store.cols)
	}
	if !resultHasIssue(res, core.SeverityWarning, "renamed") {
		t.Fatalf("missing rename warning, issues = %+v", res.Issues)
	}
}

func TestImportConflictSkip(t *testing.T) {
	store := &fakeStore{existing: []*domaincol.Collection{domaincol.New("ws-1", "API")}}
	svc := newTestService(store)

	_, err := svc.Import(context.Background(), "ws-1", []byte(postmanDoc), Options{Conflicts: ConflictSkip})
	if !errors.Is(err, ErrEmptyImport) {
		t.Fatalf("err = %v, want ErrEmptyImport", err)
	}
	if len(store.cols) != 0 {
		t.Fatalf("nothing must be persisted on skip, got %d", len(store.cols))
	}
}

func TestImportUnsupportedFormat(t *testing.T) {
	svc := newTestService(&fakeStore{})
	_, err := svc.Import(context.Background(), "ws-1", []byte(`{"nothing": true}`), Options{})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
	}
}

func TestImportWorkspaceNotFound(t *testing.T) {
	svc := New(Deps{
		Workspaces:  fakeWorkspaces{err: domainws.ErrNotFound},
		Collections: &fakeStore{},
	})
	_, err := svc.Import(context.Background(), "ws-missing", []byte(postmanDoc), Options{})
	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("err = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestImportEmptyDraft(t *testing.T) {
	svc := New(Deps{
		Workspaces:  fakeWorkspaces{ws: domainws.NewWorkspace("Main")},
		Collections: &fakeStore{},
		Importers:   []core.Importer{stubImporter{draft: &core.Draft{}}},
	})
	_, err := svc.Import(context.Background(), "ws-1", []byte("{}"), Options{})
	if !errors.Is(err, ErrEmptyImport) {
		t.Fatalf("err = %v, want ErrEmptyImport", err)
	}
}

func TestImportBatchFailurePropagates(t *testing.T) {
	store := &fakeStore{batchErr: errors.New("disk full")}
	svc := newTestService(store)

	_, err := svc.Import(context.Background(), "ws-1", []byte(postmanDoc), Options{})
	if err == nil || !strings.Contains(err.Error(), "persist import") {
		t.Fatalf("err = %v, want persist import failure", err)
	}
	if len(store.cols) != 0 {
		t.Fatalf("failed batch must not be recorded, got %d", len(store.cols))
	}
}

func TestImportIntegrationSQLite(t *testing.T) {
	ctx := context.Background()
	db, err := sqlitestore.Open(ctx, sqlitestore.Config{Path: filepath.Join(t.TempDir(), "import.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	wsr := wsrepo.New(db)
	ws := domainws.NewWorkspace("Main")
	if err := wsr.Save(ctx, ws); err != nil {
		t.Fatalf("save workspace: %v", err)
	}
	svc := New(Deps{Workspaces: wsr, Collections: colrepo.New(db)})

	res, err := svc.Import(ctx, ws.ID, []byte(postmanDoc), Options{})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.Collections != 1 || res.Folders != 1 || res.Requests != 2 {
		t.Fatalf("counts = %d/%d/%d", res.Collections, res.Folders, res.Requests)
	}

	repo := colrepo.New(db)
	cols, err := repo.ListCollections(ctx, ws.ID)
	if err != nil || len(cols) != 1 || cols[0].Name != "API" {
		t.Fatalf("collections = %+v err=%v", cols, err)
	}
	folders, err := repo.ListFolders(ctx, cols[0].ID)
	if err != nil || len(folders) != 1 || folders[0].Name != "Users" {
		t.Fatalf("folders = %+v err=%v", folders, err)
	}
	reqs, err := repo.ListRequests(ctx, ws.ID, cols[0].ID, "")
	if err != nil || len(reqs) != 2 {
		t.Fatalf("requests = %+v err=%v", reqs, err)
	}
	if reqs[0].Name != "List" || reqs[0].Method != "GET" {
		t.Fatalf("request = %+v", reqs[0])
	}
	if reqs[0].Headers["Accept"] != "application/json" || len(reqs[0].Headers) != 1 {
		t.Fatalf("persisted headers = %v", reqs[0].Headers)
	}
	if reqs[0].FolderID != folders[0].ID {
		t.Fatalf("folder linkage = %q, want %q", reqs[0].FolderID, folders[0].ID)
	}

	// Importing the same source again must rename instead of colliding.
	if _, err := svc.Import(ctx, ws.ID, []byte(postmanDoc), Options{}); err != nil {
		t.Fatalf("second import: %v", err)
	}
	cols, err = repo.ListCollections(ctx, ws.ID)
	if err != nil || len(cols) != 2 {
		t.Fatalf("collections after second import = %+v err=%v", cols, err)
	}
	names := map[string]bool{}
	for _, c := range cols {
		names[c.Name] = true
	}
	if !names["API"] || !names["API (imported)"] {
		t.Fatalf("collection names = %v", names)
	}
}

func resultHasIssue(res *Result, severity, contains string) bool {
	for _, is := range res.Issues {
		if is.Severity == severity && strings.Contains(is.Message, contains) {
			return true
		}
	}
	return false
}

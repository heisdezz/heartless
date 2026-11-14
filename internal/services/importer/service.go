// Package importer_service implements the collection import pipeline: format
// detection, parsing, validation, conflict resolution, and transactional
// persistence of collections/folders/requests into a target workspace.
//
// Importers are adapters over external documents; this service owns the
// normalization rules (default names, method/kind normalization, header and
// parameter deduplication) and the single-transaction write.
package importer_service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
	domainws "heartless/internal/domain/workspace"
	core "heartless/internal/importer"
	"heartless/internal/importer/insomnia"
	"heartless/internal/importer/postman"
)

// ConflictPolicy controls what happens when an imported collection name
// already exists in the target workspace.
type ConflictPolicy string

const (
	// ConflictRename imports the collection under a suffixed name (default).
	ConflictRename ConflictPolicy = "rename"
	// ConflictSkip drops collections whose names already exist.
	ConflictSkip ConflictPolicy = "skip"
)

// Options tunes a single Import call.
type Options struct {
	// Conflicts defaults to ConflictRename.
	Conflicts ConflictPolicy
}

// WorkspaceGetter resolves the import target workspace.
type WorkspaceGetter interface {
	Get(ctx context.Context, id string) (*domainws.Workspace, error)
}

// CollectionStore persists the normalized import batch. ImportBatch must be
// atomic: all entities are written or none are.
type CollectionStore interface {
	ListCollections(ctx context.Context, workspaceID string) ([]*domaincol.Collection, error)
	ImportBatch(ctx context.Context, collections []*domaincol.Collection, folders []*domaincol.Folder, requests []*domainreq.Request) error
}

// Result summarizes a completed import. Issues carries both the source-level
// issues reported by the adapter and normalization/conflict warnings added by
// the service.
type Result struct {
	Format      core.Format
	Collections int
	Folders     int
	Requests    int
	// CreatedIDs are the IDs of the collections created, in order.
	CreatedIDs []string
	Issues     []core.Issue
}

// Deps are the service dependencies.
type Deps struct {
	Workspaces  WorkspaceGetter
	Collections CollectionStore
	// Importers overrides the default registry (Postman, Insomnia).
	Importers []core.Importer
}

// Service runs the import pipeline.
type Service struct {
	workspaces  WorkspaceGetter
	collections CollectionStore
	importers   []core.Importer
}

var (
	// ErrUnsupportedFormat is returned when no registered importer recognizes
	// the source document.
	ErrUnsupportedFormat = core.ErrUnsupportedFormat
	// ErrWorkspaceNotFound is returned when the target workspace does not exist.
	ErrWorkspaceNotFound = errors.New("workspace not found")
	// ErrEmptyImport is returned when the source yields no importable collections.
	ErrEmptyImport = errors.New("import contains no collections")

	errWorkspaceIDRequired = errors.New("workspace id is required")
	errNotInitialized      = errors.New("importer service is not initialized")
)

// New creates an import service. When deps.Importers is empty the default
// registry (Postman, Insomnia) is used.
func New(deps Deps) *Service {
	importers := deps.Importers
	if len(importers) == 0 {
		importers = []core.Importer{postman.Importer{}, insomnia.Importer{}}
	}
	return &Service{
		workspaces:  deps.Workspaces,
		collections: deps.Collections,
		importers:   importers,
	}
}

// Formats returns the source formats this service can import.
func (s *Service) Formats() []core.Format {
	if s == nil {
		return nil
	}
	out := make([]core.Format, 0, len(s.importers))
	for _, im := range s.importers {
		out = append(out, im.Format())
	}
	return out
}

// Import parses data, maps it onto collections/folders/requests, and persists
// them into workspaceID in a single transaction. Source-level and
// normalization issues are reported in Result.Issues; a non-nil error means
// nothing was persisted.
func (s *Service) Import(ctx context.Context, workspaceID string, data []byte, opts Options) (*Result, error) {
	if s == nil || s.workspaces == nil || s.collections == nil {
		return nil, errNotInitialized
	}
	if strings.TrimSpace(workspaceID) == "" {
		return nil, errWorkspaceIDRequired
	}
	if _, err := s.workspaces.Get(ctx, workspaceID); err != nil {
		if errors.Is(err, domainws.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspaceID)
		}
		return nil, fmt.Errorf("load workspace: %w", err)
	}

	im, err := core.Select(s.importers, data)
	if err != nil {
		return nil, err
	}
	draft, err := im.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s source: %w", im.Format(), err)
	}
	if len(draft.Collections) == 0 {
		return nil, ErrEmptyImport
	}

	result := &Result{
		Format: im.Format(),
		Issues: append([]core.Issue(nil), draft.Issues...),
	}

	existing, err := s.collections.ListCollections(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list existing collections: %w", err)
	}
	used := make(map[string]bool, len(existing))
	for _, c := range existing {
		used[c.Name] = true
	}

	policy := opts.Conflicts
	if policy == "" {
		policy = ConflictRename
	}

	var (
		cols    []*domaincol.Collection
		folders []*domaincol.Folder
		reqs    []*domainreq.Request
	)

	for _, src := range draft.Collections {
		name := strings.TrimSpace(src.Name)
		if name == "" {
			name = "Imported Collection"
			result.warn("", fmt.Sprintf("collection with empty name imported as %q", name))
		}
		if used[name] {
			if policy == ConflictSkip {
				result.warn("", fmt.Sprintf("collection %q skipped: name already exists", name))
				continue
			}
			renamed := uniqueName(name, used)
			result.warn("", fmt.Sprintf("collection %q renamed to %q to avoid a name conflict", name, renamed))
			name = renamed
		}
		used[name] = true

		col := domaincol.New(workspaceID, name)
		col.Description = strings.TrimSpace(src.Description)
		col.SortOrder = len(cols)
		cols = append(cols, col)
		result.CreatedIDs = append(result.CreatedIDs, col.ID)
		result.Collections++

		// Sort order is scoped to each collection.
		mapItems(src.Items, "items", col, "", 0, result, &folders, &reqs)
	}

	if len(cols) == 0 {
		return nil, ErrEmptyImport
	}
	if err := s.collections.ImportBatch(ctx, cols, folders, reqs); err != nil {
		return nil, fmt.Errorf("persist import: %w", err)
	}
	return result, nil
}

// mapItems converts draft items into domain folders/requests, assigning sort
// order within the collection and returning the next free sort order.
func mapItems(
	items []core.Item,
	path string,
	col *domaincol.Collection,
	parentID string,
	order int,
	result *Result,
	folders *[]*domaincol.Folder,
	reqs *[]*domainreq.Request,
) int {
	for i, it := range items {
		p := fmt.Sprintf("%s[%d]", path, i)
		switch {
		case it.Folder != nil:
			name := strings.TrimSpace(it.Folder.Name)
			if name == "" {
				name = "Untitled Folder"
				result.warn(p, fmt.Sprintf("folder with empty name imported as %q", name))
			}
			folder := domaincol.NewFolder(col.ID, parentID, name)
			folder.Description = strings.TrimSpace(it.Folder.Description)
			folder.SortOrder = order
			order++
			*folders = append(*folders, folder)
			result.Folders++
			order = mapItems(it.Children, p, col, folder.ID, order, result, folders, reqs)
		case it.Request != nil:
			*reqs = append(*reqs, mapRequest(it.Request, p, col, parentID, order, result))
			order++
			result.Requests++
		default:
			result.warn(p, "empty item skipped")
		}
	}
	return order
}

// mapRequest normalizes one draft request onto the domain request model.
func mapRequest(src *core.Request, path string, col *domaincol.Collection, folderID string, order int, result *Result) *domainreq.Request {
	name := strings.TrimSpace(src.Name)
	if name == "" {
		name = "Untitled Request"
		result.warn(path, fmt.Sprintf("request with empty name imported as %q", name))
	}

	kind := domainreq.Kind(strings.TrimSpace(src.Kind))
	if kind == "" {
		kind = domainreq.KindHTTP
	}
	switch kind {
	case domainreq.KindHTTP, domainreq.KindGraphQL, domainreq.KindGRPC, domainreq.KindWebSocket:
	default:
		result.warn(path, fmt.Sprintf("unknown request kind %q imported as http", kind))
		kind = domainreq.KindHTTP
	}

	method := strings.ToUpper(strings.TrimSpace(src.Method))
	if method == "" {
		method = "GET"
	}

	req := domainreq.New(col.WorkspaceID, name, kind)
	req.CollectionID = col.ID
	req.FolderID = folderID
	req.Description = strings.TrimSpace(src.Description)
	req.Method = method
	req.URL = strings.TrimSpace(src.URL)
	req.Body = src.Body
	req.BodyType = src.BodyType
	req.SortOrder = order
	req.Headers = nvpMap(src.Headers, "header", path, result)
	req.Params = nvpMap(src.Params, "parameter", path, result)
	return req
}

// nvpMap converts enabled name/value pairs into the map shape the domain
// request model uses. Empty names are dropped with a warning; the first
// occurrence of a duplicate name wins.
func nvpMap(list []core.NameValue, label, path string, result *Result) map[string]string {
	m := make(map[string]string, len(list))
	for _, nv := range list {
		if !nv.Enabled {
			continue
		}
		name := strings.TrimSpace(nv.Name)
		if name == "" {
			result.warn(path, fmt.Sprintf("%s with empty name skipped", label))
			continue
		}
		if _, dup := m[name]; dup {
			result.warn(path, fmt.Sprintf("duplicate %s %q: kept first value", label, name))
			continue
		}
		m[name] = nv.Value
	}
	return m
}

// uniqueName returns base or a numbered variant not present in used.
func uniqueName(base string, used map[string]bool) string {
	candidate := base + " (imported)"
	for i := 2; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s (imported %d)", base, i)
	}
	return candidate
}

func (r *Result) warn(path, message string) {
	r.Issues = append(r.Issues, core.Issue{
		Severity: core.SeverityWarning,
		Path:     path,
		Message:  message,
	})
}

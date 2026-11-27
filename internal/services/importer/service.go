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
	// ConflictMerge updates matching existing collections in place, preserving
	// existing request IDs and post-response scripts for requests that remain
	// while removing requests and folders no longer present in the import.
	ConflictMerge ConflictPolicy = "merge"
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

// CollectionStore persists the normalized import batch. ImportBatch and
// SyncBatch must be atomic: all entities are written/deleted or none are.
type CollectionStore interface {
	ListCollections(ctx context.Context, workspaceID string) ([]*domaincol.Collection, error)
	ListFolders(ctx context.Context, collectionID string) ([]*domaincol.Folder, error)
	ListRequests(ctx context.Context, workspaceID, collectionID, folderID string) ([]*domainreq.Request, error)
	ImportBatch(ctx context.Context, collections []*domaincol.Collection, folders []*domaincol.Folder, requests []*domainreq.Request) error
	SyncBatch(ctx context.Context, collections []*domaincol.Collection, folders []*domaincol.Folder, requests []*domainreq.Request, deleteFolderIDs, deleteRequestIDs []string) error
}

// Inspection summarizes an import source and any collection name conflicts
// against the target workspace before persisting changes.
type Inspection struct {
	Format              core.Format
	Collections         []string
	ExistingCollections []string
}

// Result summarizes a completed import. Issues carries both the source-level
// issues reported by the adapter and normalization/conflict warnings added by
// the service.
type Result struct {
	Format          core.Format
	Collections     int
	Folders         int
	Requests        int
	DeletedFolders  int
	DeletedRequests int
	// CreatedIDs are the IDs of the collections created, in order.
	CreatedIDs []string
	// MergedIDs are the IDs of existing collections updated by a merge, in order.
	MergedIDs         []string
	DeletedRequestIDs []string
	Issues            []core.Issue
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

// Inspect validates and parses data and reports which imported collections
// already exist by name in workspaceID, without writing anything.
func (s *Service) Inspect(ctx context.Context, workspaceID string, data []byte) (*Inspection, error) {
	im, draft, existing, err := s.prepare(ctx, workspaceID, data)
	if err != nil {
		return nil, err
	}
	used := make(map[string]bool, len(existing))
	for _, c := range existing {
		if c != nil {
			used[c.Name] = true
		}
	}
	seenConflict := make(map[string]bool)
	info := &Inspection{
		Format:      im.Format(),
		Collections: make([]string, 0, len(draft.Collections)),
	}
	for _, src := range draft.Collections {
		name := normalizeCollectionName(src.Name)
		info.Collections = append(info.Collections, name)
		if used[name] && !seenConflict[name] {
			seenConflict[name] = true
			info.ExistingCollections = append(info.ExistingCollections, name)
		}
	}
	return info, nil
}

// Import parses data, maps it onto collections/folders/requests, and persists
// them into workspaceID in a single transaction. Source-level and
// normalization issues are reported in Result.Issues; a non-nil error means
// nothing was persisted.
func (s *Service) Import(ctx context.Context, workspaceID string, data []byte, opts Options) (*Result, error) {
	im, draft, existing, err := s.prepare(ctx, workspaceID, data)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Format: im.Format(),
		Issues: append([]core.Issue(nil), draft.Issues...),
	}

	used := make(map[string]bool, len(existing))
	existingByName := make(map[string]*domaincol.Collection, len(existing))
	for _, c := range existing {
		if c == nil {
			continue
		}
		used[c.Name] = true
		if _, ok := existingByName[c.Name]; !ok {
			existingByName[c.Name] = c
		}
	}

	policy := opts.Conflicts
	if policy == "" {
		policy = ConflictRename
	}

	var (
		cols             []*domaincol.Collection
		folders          []*domaincol.Folder
		reqs             []*domainreq.Request
		deleteFolderIDs  []string
		deleteRequestIDs []string
		mergedNames      = make(map[string]bool)
	)

	for _, src := range draft.Collections {
		name := strings.TrimSpace(src.Name)
		if name == "" {
			name = "Imported Collection"
			result.warn("", fmt.Sprintf("collection with empty name imported as %q", name))
		}
		if used[name] {
			switch policy {
			case ConflictSkip:
				result.warn("", fmt.Sprintf("collection %q skipped: name already exists", name))
				continue
			case ConflictMerge:
				if existingCol := existingByName[name]; existingCol != nil && !mergedNames[name] {
					mergedNames[name] = true
					if err := s.mergeCollection(ctx, workspaceID, existingCol, src, result, &cols, &folders, &reqs, &deleteFolderIDs, &deleteRequestIDs); err != nil {
						return nil, err
					}
					continue
				}
				fallthrough
			default:
				renamed := uniqueName(name, used)
				result.warn("", fmt.Sprintf("collection %q renamed to %q to avoid a name conflict", name, renamed))
				name = renamed
			}
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
	if len(result.MergedIDs) > 0 || len(deleteFolderIDs) > 0 || len(deleteRequestIDs) > 0 {
		if err := s.collections.SyncBatch(ctx, cols, folders, reqs, deleteFolderIDs, deleteRequestIDs); err != nil {
			return nil, fmt.Errorf("persist import: %w", err)
		}
	} else {
		if err := s.collections.ImportBatch(ctx, cols, folders, reqs); err != nil {
			return nil, fmt.Errorf("persist import: %w", err)
		}
	}
	return result, nil
}

func (s *Service) prepare(ctx context.Context, workspaceID string, data []byte) (core.Importer, *core.Draft, []*domaincol.Collection, error) {
	if s == nil || s.workspaces == nil || s.collections == nil {
		return nil, nil, nil, errNotInitialized
	}
	if strings.TrimSpace(workspaceID) == "" {
		return nil, nil, nil, errWorkspaceIDRequired
	}
	if _, err := s.workspaces.Get(ctx, workspaceID); err != nil {
		if errors.Is(err, domainws.ErrNotFound) {
			return nil, nil, nil, fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspaceID)
		}
		return nil, nil, nil, fmt.Errorf("load workspace: %w", err)
	}

	im, err := core.Select(s.importers, data)
	if err != nil {
		return nil, nil, nil, err
	}
	draft, err := im.Parse(data)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse %s source: %w", im.Format(), err)
	}
	if len(draft.Collections) == 0 {
		return nil, nil, nil, ErrEmptyImport
	}

	existing, err := s.collections.ListCollections(ctx, workspaceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list existing collections: %w", err)
	}
	return im, draft, existing, nil
}

func normalizeCollectionName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "Imported Collection"
	}
	return name
}

// mergeCollection updates existingCol to match src while preserving existing
// folder/request IDs and set post-response scripts on kept requests. Requests
// and folders no longer present in src are scheduled for deletion.
func (s *Service) mergeCollection(
	ctx context.Context,
	workspaceID string,
	existingCol *domaincol.Collection,
	src core.Collection,
	result *Result,
	cols *[]*domaincol.Collection,
	folders *[]*domaincol.Folder,
	reqs *[]*domainreq.Request,
	deleteFolderIDs *[]string,
	deleteRequestIDs *[]string,
) error {
	existingFolders, err := s.collections.ListFolders(ctx, existingCol.ID)
	if err != nil {
		return fmt.Errorf("list existing folders for %q: %w", existingCol.Name, err)
	}
	existingReqs, err := s.collections.ListRequests(ctx, workspaceID, existingCol.ID, "")
	if err != nil {
		return fmt.Errorf("list existing requests for %q: %w", existingCol.Name, err)
	}

	mergedCol := *existingCol
	mergedCol.Description = strings.TrimSpace(src.Description)
	*cols = append(*cols, &mergedCol)
	result.MergedIDs = append(result.MergedIDs, mergedCol.ID)
	result.Collections++

	claimedFolders := make(map[string]bool, len(existingFolders))
	var incomingReqs []*domainreq.Request
	mapMergeItems(src.Items, "items", &mergedCol, "", 0, result, existingFolders, claimedFolders, folders, &incomingReqs)

	claimedReqs := make(map[string]bool, len(existingReqs))
	matched := make([]bool, len(incomingReqs))

	// Pass 1: exact match within the same folder (folder, name, kind, method).
	for i, req := range incomingReqs {
		for _, ex := range existingReqs {
			if ex == nil || claimedReqs[ex.ID] {
				continue
			}
			if ex.FolderID == req.FolderID && ex.Name == req.Name && ex.Kind == req.Kind && ex.Method == req.Method {
				applyMatchedRequest(req, ex)
				claimedReqs[ex.ID] = true
				matched[i] = true
				break
			}
		}
	}

	// Pass 2: fallback match within the same folder by name (handles method/kind edits).
	for i, req := range incomingReqs {
		if matched[i] {
			continue
		}
		for _, ex := range existingReqs {
			if ex == nil || claimedReqs[ex.ID] {
				continue
			}
			if ex.FolderID == req.FolderID && ex.Name == req.Name {
				applyMatchedRequest(req, ex)
				claimedReqs[ex.ID] = true
				matched[i] = true
				break
			}
		}
	}

	// Pass 3: moved request match across folders when name, method, and URL match.
	for i, req := range incomingReqs {
		if matched[i] || req.URL == "" {
			continue
		}
		for _, ex := range existingReqs {
			if ex == nil || claimedReqs[ex.ID] {
				continue
			}
			if ex.Name == req.Name && ex.Method == req.Method && ex.URL == req.URL {
				applyMatchedRequest(req, ex)
				claimedReqs[ex.ID] = true
				matched[i] = true
				break
			}
		}
	}

	*reqs = append(*reqs, incomingReqs...)

	for _, f := range existingFolders {
		if f != nil && !claimedFolders[f.ID] {
			*deleteFolderIDs = append(*deleteFolderIDs, f.ID)
			result.DeletedFolders++
		}
	}
	for _, r := range existingReqs {
		if r != nil && !claimedReqs[r.ID] {
			*deleteRequestIDs = append(*deleteRequestIDs, r.ID)
			result.DeletedRequestIDs = append(result.DeletedRequestIDs, r.ID)
			result.DeletedRequests++
		}
	}
	return nil
}

// applyMatchedRequest keeps the existing request's identity, creation time,
// post-response script, and auth (when the import provides none) while updating
// the imported definition fields.
func applyMatchedRequest(incoming, existing *domainreq.Request) {
	incoming.ID = existing.ID
	incoming.WorkspaceID = existing.WorkspaceID
	incoming.CollectionID = existing.CollectionID
	incoming.CreatedAt = existing.CreatedAt
	incoming.Script = existing.Script
	if incoming.Auth.IsZero() && !existing.Auth.IsZero() {
		incoming.Auth = existing.Auth
	}
}

func mapMergeItems(
	items []core.Item,
	path string,
	col *domaincol.Collection,
	parentID string,
	order int,
	result *Result,
	existingFolders []*domaincol.Folder,
	claimedFolders map[string]bool,
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
			var folder *domaincol.Folder
			for _, ex := range existingFolders {
				if ex == nil || claimedFolders[ex.ID] {
					continue
				}
				if ex.ParentID == parentID && ex.Name == name {
					copyFolder := *ex
					copyFolder.Description = strings.TrimSpace(it.Folder.Description)
					copyFolder.SortOrder = order
					claimedFolders[ex.ID] = true
					folder = &copyFolder
					break
				}
			}
			if folder == nil {
				folder = domaincol.NewFolder(col.ID, parentID, name)
				folder.Description = strings.TrimSpace(it.Folder.Description)
				folder.SortOrder = order
			}
			order++
			*folders = append(*folders, folder)
			result.Folders++
			order = mapMergeItems(it.Children, p, col, folder.ID, order, result, existingFolders, claimedFolders, folders, reqs)
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

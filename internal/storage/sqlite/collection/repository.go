package collection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "heartless/internal/domain/collection"
	"heartless/internal/domain/request"
)

// execer is the subset of database operations shared by *sql.DB and *sql.Tx,
// so save helpers can run standalone or inside a transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Repository provides SQLite persistence for collections, folders, and requests.
type Repository struct {
	db *sql.DB
}

// New creates a collection repository backed by SQLite.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetCollection(ctx context.Context, id string) (*domain.Collection, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, description, sort_order, created_at, updated_at FROM collections WHERE id = ?`, id)
	return scanCollection(row)
}

func (r *Repository) SaveCollection(ctx context.Context, col *domain.Collection) error {
	return saveCollection(ctx, r.db, col)
}

// saveCollection writes a collection through ex, which may be a transaction.
func saveCollection(ctx context.Context, ex execer, col *domain.Collection) error {
	if col == nil {
		return errors.New("collection is nil")
	}
	if col.CreatedAt.IsZero() {
		col.CreatedAt = time.Now().UTC()
	}
	col.UpdatedAt = time.Now().UTC()
	_, err := ex.ExecContext(ctx, `
INSERT INTO collections (id, workspace_id, name, description, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	workspace_id = excluded.workspace_id,
	name = excluded.name,
	description = excluded.description,
	sort_order = excluded.sort_order,
	updated_at = excluded.updated_at
`, col.ID, col.WorkspaceID, col.Name, col.Description, col.SortOrder, formatTime(col.CreatedAt), formatTime(col.UpdatedAt))
	return err
}

func (r *Repository) DeleteCollection(ctx context.Context, id string) error {
	return r.delete(ctx, "DELETE FROM collections WHERE id = ?", id, domain.ErrCollectionNotFound)
}

func (r *Repository) ListCollections(ctx context.Context, workspaceID string) ([]*domain.Collection, error) {
	query := `SELECT id, workspace_id, name, description, sort_order, created_at, updated_at FROM collections`
	args := []any{}
	if workspaceID != "" {
		query += ` WHERE workspace_id = ?`
		args = append(args, workspaceID)
	}
	query += ` ORDER BY sort_order ASC, name ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Collection
	for rows.Next() {
		col, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, col)
	}
	return out, rows.Err()
}

func (r *Repository) GetFolder(ctx context.Context, id string) (*domain.Folder, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, collection_id, parent_id, name, description, sort_order, created_at, updated_at FROM folders WHERE id = ?`, id)
	return scanFolder(row)
}

func (r *Repository) SaveFolder(ctx context.Context, folder *domain.Folder) error {
	return saveFolder(ctx, r.db, folder)
}

// saveFolder writes a folder through ex, which may be a transaction.
func saveFolder(ctx context.Context, ex execer, folder *domain.Folder) error {
	if folder == nil {
		return errors.New("folder is nil")
	}
	if folder.CreatedAt.IsZero() {
		folder.CreatedAt = time.Now().UTC()
	}
	folder.UpdatedAt = time.Now().UTC()
	_, err := ex.ExecContext(ctx, `
INSERT INTO folders (id, collection_id, parent_id, name, description, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	collection_id = excluded.collection_id,
	parent_id = excluded.parent_id,
	name = excluded.name,
	description = excluded.description,
	sort_order = excluded.sort_order,
	updated_at = excluded.updated_at
`, folder.ID, folder.CollectionID, folder.ParentID, folder.Name, folder.Description, folder.SortOrder, formatTime(folder.CreatedAt), formatTime(folder.UpdatedAt))
	return err
}

func (r *Repository) DeleteFolder(ctx context.Context, id string) error {
	return r.delete(ctx, "DELETE FROM folders WHERE id = ?", id, domain.ErrFolderNotFound)
}

func (r *Repository) ListFolders(ctx context.Context, collectionID string) ([]*domain.Folder, error) {
	query := `SELECT id, collection_id, parent_id, name, description, sort_order, created_at, updated_at FROM folders`
	args := []any{}
	if collectionID != "" {
		query += ` WHERE collection_id = ?`
		args = append(args, collectionID)
	}
	query += ` ORDER BY sort_order ASC, name ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Folder
	for rows.Next() {
		folder, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, folder)
	}
	return out, rows.Err()
}

func (r *Repository) GetRequest(ctx context.Context, id string) (*request.Request, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, collection_id, folder_id, name, description, kind, method, url, body, body_type, script, headers, params, auth, sort_order, created_at, updated_at FROM requests WHERE id = ?`, id)
	return scanRequest(row)
}

func (r *Repository) SaveRequest(ctx context.Context, req *request.Request) error {
	return saveRequest(ctx, r.db, req)
}

// saveRequest writes a saved request (including headers and params as JSON)
// through ex, which may be a transaction.
func saveRequest(ctx context.Context, ex execer, req *request.Request) error {
	if req == nil {
		return errors.New("request is nil")
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	req.UpdatedAt = time.Now().UTC()
	headers, err := encodeNV(req.Headers)
	if err != nil {
		return fmt.Errorf("encode request headers: %w", err)
	}
	params, err := encodeNV(req.Params)
	if err != nil {
		return fmt.Errorf("encode request params: %w", err)
	}
	auth := "{}"
	if !req.Auth.IsZero() {
		b, err := json.Marshal(req.Auth)
		if err != nil {
			return fmt.Errorf("encode request auth: %w", err)
		}
		auth = string(b)
	}
	_, err = ex.ExecContext(ctx, `
INSERT INTO requests (id, workspace_id, collection_id, folder_id, name, description, kind, method, url, body, body_type, script, headers, params, auth, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	workspace_id = excluded.workspace_id,
	collection_id = excluded.collection_id,
	folder_id = excluded.folder_id,
	name = excluded.name,
	description = excluded.description,
	kind = excluded.kind,
	method = excluded.method,
	url = excluded.url,
	body = excluded.body,
	body_type = excluded.body_type,
	script = excluded.script,
	headers = excluded.headers,
	params = excluded.params,
	auth = excluded.auth,
	sort_order = excluded.sort_order,
	updated_at = excluded.updated_at
`, req.ID, req.WorkspaceID, req.CollectionID, req.FolderID, req.Name, req.Description, string(req.Kind), req.Method, req.URL, req.Body, req.BodyType, req.Script, headers, params, auth, req.SortOrder, formatTime(req.CreatedAt), formatTime(req.UpdatedAt))
	return err
}

func (r *Repository) DeleteRequest(ctx context.Context, id string) error {
	return r.delete(ctx, "DELETE FROM requests WHERE id = ?", id, domain.ErrRequestNotFound)
}

func (r *Repository) ListRequests(ctx context.Context, workspaceID, collectionID, folderID string) ([]*request.Request, error) {
	query := `SELECT id, workspace_id, collection_id, folder_id, name, description, kind, method, url, body, body_type, script, headers, params, auth, sort_order, created_at, updated_at FROM requests WHERE 1=1`
	args := []any{}
	if workspaceID != "" {
		query += ` AND workspace_id = ?`
		args = append(args, workspaceID)
	}
	if collectionID != "" {
		query += ` AND collection_id = ?`
		args = append(args, collectionID)
	}
	if folderID != "" {
		query += ` AND folder_id = ?`
		args = append(args, folderID)
	}
	query += ` ORDER BY sort_order ASC, name ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*request.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (r *Repository) delete(ctx context.Context, query, id string, notFound error) error {
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCollection(s scanner) (*domain.Collection, error) {
	var col domain.Collection
	var createdAt, updatedAt string
	if err := s.Scan(&col.ID, &col.WorkspaceID, &col.Name, &col.Description, &col.SortOrder, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, err
	}
	col.CreatedAt = parseTime(createdAt)
	col.UpdatedAt = parseTime(updatedAt)
	return &col, nil
}

func scanFolder(s scanner) (*domain.Folder, error) {
	var folder domain.Folder
	var createdAt, updatedAt string
	if err := s.Scan(&folder.ID, &folder.CollectionID, &folder.ParentID, &folder.Name, &folder.Description, &folder.SortOrder, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrFolderNotFound
		}
		return nil, err
	}
	folder.CreatedAt = parseTime(createdAt)
	folder.UpdatedAt = parseTime(updatedAt)
	return &folder, nil
}

func scanRequest(s scanner) (*request.Request, error) {
	var req request.Request
	var kind, headers, params, authJSON string
	var createdAt, updatedAt string
	if err := s.Scan(&req.ID, &req.WorkspaceID, &req.CollectionID, &req.FolderID, &req.Name, &req.Description, &kind, &req.Method, &req.URL, &req.Body, &req.BodyType, &req.Script, &headers, &params, &authJSON, &req.SortOrder, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRequestNotFound
		}
		return nil, err
	}
	decoded, err := decodeNV(headers)
	if err != nil {
		return nil, fmt.Errorf("request %s headers: %w", req.ID, err)
	}
	req.Headers = decoded
	decoded, err = decodeNV(params)
	if err != nil {
		return nil, fmt.Errorf("request %s params: %w", req.ID, err)
	}
	req.Params = decoded
	if authJSON != "" && authJSON != "{}" {
		if err := json.Unmarshal([]byte(authJSON), &req.Auth); err != nil {
			return nil, fmt.Errorf("request %s auth: %w", req.ID, err)
		}
	}
	req.Kind = request.Kind(kind)
	req.CreatedAt = parseTime(createdAt)
	req.UpdatedAt = parseTime(updatedAt)
	return &req, nil
}

// encodeNV serializes a name/value map for storage; empty maps become "{}".
func encodeNV(m map[string]string) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeNV parses a stored name/value map, tolerating empty input.
func decodeNV(s string) (map[string]string, error) {
	m := make(map[string]string)
	if strings.TrimSpace(s) == "" {
		return m, nil
	}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]string)
	}
	return m, nil
}

const timeLayout = "2006-01-02T15:04:05Z"

func formatTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

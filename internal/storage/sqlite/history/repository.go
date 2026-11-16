package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	domain "heartless/internal/domain/history"
	"heartless/internal/domain/response"
)

var ErrNotFound = errors.New("history entry not found")

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Record persists an execution history entry (alias for Save, satisfies recorder interfaces).
func (r *Repository) Record(ctx context.Context, entry *domain.Entry) error {
	return r.Save(ctx, entry)
}

func (r *Repository) Save(ctx context.Context, entry *domain.Entry) error {
	if entry == nil {
		return errors.New("history entry is nil")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	statusText, headers, body, contentType, size := responseFields(entry.Response)
	resID, resRequestID, resDurationNS, effectiveURL := responseMeta(entry.Response)
	_, err := r.db.ExecContext(ctx, `
INSERT INTO execution_history (id, workspace_id, request_id, method, url, status_code, duration_ms, error, created_at,
	status_text, response_headers, body, content_type, size, response_id, response_request_id, response_duration_ns, effective_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	workspace_id = excluded.workspace_id,
	request_id = excluded.request_id,
	method = excluded.method,
	url = excluded.url,
	status_code = excluded.status_code,
	duration_ms = excluded.duration_ms,
	error = excluded.error,
	status_text = excluded.status_text,
	response_headers = excluded.response_headers,
	body = excluded.body,
	content_type = excluded.content_type,
	size = excluded.size,
	response_id = excluded.response_id,
	response_request_id = excluded.response_request_id,
	response_duration_ns = excluded.response_duration_ns,
	effective_url = excluded.effective_url
`, entry.ID, entry.WorkspaceID, entry.RequestID, entry.Method, entry.URL, entry.StatusCode, entry.DurationMS, entry.Error, formatTime(entry.CreatedAt),
		statusText, headers, body, contentType, size, resID, resRequestID, resDurationNS, effectiveURL)
	return err
}

// responseMeta extracts the remaining response metadata columns. Duration is
// stored in nanoseconds so short round-trips are not truncated to zero.
func responseMeta(res *response.Response) (resID, requestID string, durationNS int64, effectiveURL string) {
	if res == nil {
		return "", "", 0, ""
	}
	return res.ID, res.RequestID, res.Duration.Nanoseconds(), res.EffectiveURL
}

// responseFields extracts the persisted columns from a response, encoding its
// headers as JSON.
func responseFields(res *response.Response) (statusText, headers, body, contentType string, size int64) {
	headers = "{}"
	if res == nil {
		return statusText, headers, body, contentType, 0
	}
	if res.Headers != nil {
		if b, err := json.Marshal(res.Headers); err == nil {
			headers = string(b)
		}
	}
	return res.StatusText, headers, string(res.Body), res.ContentType, res.Size
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Entry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+historyColumns+` FROM execution_history WHERE id = ?`, id)
	return scan(row)
}

// RequestIDsWithResponses returns the distinct request IDs that have at least
// one history entry with a persisted response body in a workspace. It selects
// only the ID column, so it stays cheap even when bodies are large.
func (r *Repository) RequestIDsWithResponses(ctx context.Context, workspaceID string) ([]string, error) {
	q := `SELECT DISTINCT request_id FROM execution_history
		WHERE request_id != '' AND (body != '' OR status_code != 0 OR response_id != '' OR content_type != '')`
	args := []any{}
	if workspaceID != "" {
		q += ` AND workspace_id = ?`
		args = append(args, workspaceID)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LatestByRequestID returns the most recent history entry recorded for a
// request, including its persisted response payload. request_id is indexed, so
// this is cheap even with a large history table.
func (r *Repository) LatestByRequestID(ctx context.Context, requestID string) (*domain.Entry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+historyColumns+` FROM execution_history WHERE request_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`,
		requestID)
	return scan(row)
}

const historyColumns = `id, workspace_id, request_id, method, url, status_code, duration_ms, error, created_at, status_text, response_headers, body, content_type, size, response_id, response_request_id, response_duration_ns, effective_url`

func (r *Repository) List(ctx context.Context, workspaceID string, limit int) ([]*domain.Entry, error) {
	q := `SELECT ` + historyColumns + ` FROM execution_history`
	args := []any{}
	if workspaceID != "" {
		q += ` WHERE workspace_id = ?`
		args = append(args, workspaceID)
	}
	q += ` ORDER BY created_at DESC, id DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Entry
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM execution_history WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Clear(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		_, err := r.db.ExecContext(ctx, `DELETE FROM execution_history`)
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM execution_history WHERE workspace_id = ?`, workspaceID)
	return err
}

// Prune keeps the newest limit entries for a workspace and deletes older ones.
func (r *Repository) Prune(ctx context.Context, workspaceID string, limit int) error {
	if limit <= 0 {
		return r.Clear(ctx, workspaceID)
	}
	if workspaceID == "" {
		_, err := r.db.ExecContext(ctx, `
DELETE FROM execution_history
WHERE id NOT IN (
	SELECT id FROM execution_history ORDER BY created_at DESC, id DESC LIMIT ?
)`, limit)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
DELETE FROM execution_history
WHERE workspace_id = ? AND id NOT IN (
	SELECT id FROM execution_history WHERE workspace_id = ? ORDER BY created_at DESC, id DESC LIMIT ?
)`, workspaceID, workspaceID, limit)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(s scanner) (*domain.Entry, error) {
	var e domain.Entry
	var createdAt, headersJSON string
	var statusText, body, contentType string
	var size, resDurationNS int64
	var resID, resRequestID, effectiveURL string
	if err := s.Scan(&e.ID, &e.WorkspaceID, &e.RequestID, &e.Method, &e.URL, &e.StatusCode, &e.DurationMS, &e.Error, &createdAt,
		&statusText, &headersJSON, &body, &contentType, &size, &resID, &resRequestID, &resDurationNS, &effectiveURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	if statusText != "" || headersJSON != "" || body != "" || contentType != "" || size != 0 || resID != "" || effectiveURL != "" {
		res := &response.Response{
			ID:           resID,
			RequestID:    resRequestID,
			StatusCode:   e.StatusCode,
			StatusText:   statusText,
			Body:         []byte(body),
			ContentType:  contentType,
			Size:         size,
			Duration:     time.Duration(resDurationNS),
			EffectiveURL: effectiveURL,
			CreatedAt:    e.CreatedAt,
		}
		if headersJSON != "" {
			var h map[string]string
			if err := json.Unmarshal([]byte(headersJSON), &h); err == nil {
				res.Headers = h
			}
		}
		if e.Error != "" {
			res.Error = e.Error
		}
		e.Response = res
	}
	return &e, nil
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

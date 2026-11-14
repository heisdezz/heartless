package history

import (
	"time"

	"heartless/internal/domain/id"
	"heartless/internal/domain/response"
)

// Entry records a persisted request execution in history.
type Entry struct {
	ID          string              `json:"id" db:"id"`
	WorkspaceID string              `json:"workspace_id" db:"workspace_id"`
	RequestID   string              `json:"request_id,omitempty" db:"request_id"`
	Method      string              `json:"method,omitempty" db:"method"`
	URL         string              `json:"url,omitempty" db:"url"`
	StatusCode  int                 `json:"status_code,omitempty" db:"status_code"`
	DurationMS  int64               `json:"duration_ms,omitempty" db:"duration_ms"`
	Error       string              `json:"error,omitempty" db:"error"`
	Response    *response.Response  `json:"response,omitempty" db:"-"`
	CreatedAt   time.Time           `json:"created_at" db:"created_at"`
}

// New creates a history entry with a generated ID and timestamp.
func New(workspaceID string) *Entry {
	return &Entry{
		ID:          id.New("hist"),
		WorkspaceID: workspaceID,
		CreatedAt:   time.Now().UTC(),
	}
}

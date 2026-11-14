package workspace

import (
	"time"

	"heartless/internal/domain/id"
)

// Workspace represents a top-level container for API work.
type Workspace struct {
	ID                  string    `json:"id" db:"id"`
	Name                string    `json:"name" db:"name"`
	Description         string    `json:"description,omitempty" db:"description"`
	ActiveEnvironmentID string    `json:"active_environment_id,omitempty" db:"active_environment_id"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// New creates a workspace with a generated ID and timestamps.
func New(name string) *Workspace {
	now := time.Now().UTC()
	return &Workspace{
		ID:        id.New("ws"),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewWorkspace is a compatibility alias for New.
func NewWorkspace(name string) *Workspace { return New(name) }

var ErrNotFound = workspaceError("not found")

type workspaceError string

func (e workspaceError) Error() string { return string(e) }

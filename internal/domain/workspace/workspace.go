package workspace

import "time"

// Workspace represents a top-level container for API work.
// Workspaces organize collections, environments, requests, and settings.
type Workspace struct {
	// ID is the unique identifier for the workspace.
	ID string `json:"id" db:"id"`

	// Name is the display name of the workspace.
	Name string `json:"name" db:"name"`

	// Description provides human-readable description of the workspace.
	Description string `json:"description" db:"description"`

	// CreatedAt tracks when the workspace was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt tracks when the workspace was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ErrNotFound is returned when a workspace is not found.
var ErrNotFound = workspaceError("not found")

type workspaceError string

func (e workspaceError) Error() string {
	return string(e)
}

// NewWorkspace creates a new workspace with the given name.
func NewWorkspace(name string) *Workspace {
	return &Workspace{
		ID:          generateID(),
		Name:        name,
		Description: "",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// IsEmpty returns true if the workspace has no name.
func (w *Workspace) IsEmpty() bool {
	return w.Name == ""
}

func generateID() string {
	// Simple ID generation; in production this would use a proper UUID generator.
	return "ws-" // placeholder
}
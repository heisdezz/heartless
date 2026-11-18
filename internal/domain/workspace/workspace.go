package workspace

import (
	"encoding/json"
	"heartless/internal/domain/id"
	domainreq "heartless/internal/domain/request"
	"time"
)

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

	// DefaultHeaders are headers automatically added to all requests in this workspace.
	DefaultHeaders map[string]string `json:"default_headers,omitempty" db:"default_headers"`

	// DefaultAuth is the default authentication applied to all requests in this workspace.
	DefaultAuth domainreq.Auth `json:"default_auth,omitempty" db:"default_auth"`

	// AutoInheritAuth applies DefaultAuth to requests that carry no auth of their
	// own, instead of leaving them unauthenticated.
	AutoInheritAuth bool `json:"auto_inherit_auth,omitempty" db:"auto_inherit_auth"`

	// DefaultTLSInsecure skips TLS verification for all requests in this workspace.
	DefaultTLSInsecure bool `json:"default_tls_insecure,omitempty" db:"default_tls_insecure"`
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
		ID:                 generateID(),
		Name:               name,
		Description:        "",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		DefaultHeaders:     map[string]string{},
		DefaultAuth:        domainreq.Auth{},
		DefaultTLSInsecure: false,
	}
}

// IsEmpty returns true if the workspace has no name.
func (w *Workspace) IsEmpty() bool {
	return w.Name == ""
}

func generateID() string {
	return id.New("ws")
}

// DefaultHeaders is a map of header name -> value that are automatically
// added to all requests in this workspace.
type DefaultHeaders map[string]string

// UnmarshalJSON implements the json.Unmarshaler interface for DefaultHeaders.
func (d *DefaultHeaders) UnmarshalJSON(data []byte) error {
	// Simple JSON parsing for a map - manually decode
	var rawMap map[string]string
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}
	*d = rawMap
	return nil
}

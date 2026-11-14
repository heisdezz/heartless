package workspace

import (
	"encoding/json"
	"heartless/internal/domain/id"
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
	DefaultAuth Auth `json:"default_auth,omitempty" db:"default_auth"`

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
		DefaultAuth:        Auth{},
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

// Auth holds a workspace's default authentication settings.
type Auth struct {
	// Type identifies how a request authenticates.
	Type string `json:"type,omitempty" db:"type"`

	// Username and Password are used by Basic and Digest auth.
	Username string `json:"username,omitempty" db:"username"`
	Password string `json:"password,omitempty" db:"password"`

	// Key, Header and Prefix are used by API key auth.
	Key    string `json:"key,omitempty" db:"key"`
	Header string `json:"header,omitempty" db:"header"`
	Prefix string `json:"prefix,omitempty" db:"prefix"`

	// Oauth1 and OAuth2 settings.
	OAuth1Token string `json:"oauth1_token,omitempty" db:"oauth1_token"`
	OAuth2Token string `json:"oauth2_token,omitempty" db:"oauth2_token"`
	OAuth2Scope string `json:"oauth2_scope,omitempty" db:"oauth2_scope"`
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

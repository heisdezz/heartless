package variable

import (
	"time"

	"heartless/internal/domain/id"
)

// Variable represents a reusable value that can be interpolated in requests.
type Variable struct {
	ID            string    `json:"id" db:"id"`
	WorkspaceID   string    `json:"workspace_id,omitempty" db:"workspace_id"`
	EnvironmentID string    `json:"environment_id,omitempty" db:"environment_id"`
	Name          string    `json:"name" db:"name"`
	Value         string    `json:"value" db:"value"`
	Type          string    `json:"type,omitempty" db:"type"`
	Description   string    `json:"description,omitempty" db:"description"`
	Secret        bool      `json:"secret,omitempty" db:"secret"`
	Scope         string    `json:"scope,omitempty" db:"scope"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// New creates a variable with a generated ID and timestamps.
func New(name, value string) *Variable {
	now := time.Now().UTC()
	return &Variable{
		ID:        id.New("var"),
		Name:      name,
		Value:     value,
		Type:      "text",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewVariable is a compatibility alias for New.
func NewVariable(name, value, typ string, secret bool) *Variable {
	v := New(name, value)
	if typ != "" {
		v.Type = typ
	}
	v.Secret = secret
	if v.Scope == "" {
		v.Scope = "request"
	}
	return v
}

var ErrNotFound = variableError("not found")

type variableError string

func (e variableError) Error() string { return string(e) }

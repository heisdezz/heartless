package variable

import "time"

// Variable represents a reusable value that can be interpolated in requests.
// Variables support scoping and type information for proper resolution and secret handling.
type Variable struct {
	// ID is the unique identifier for the variable.
	ID string `json:"id" db:"id"`

	// WorkspaceID associates the variable with a workspace (optional scope).
	WorkspaceID string `json:"workspace_id,omitempty" db:"workspace_id"`

	// EnvironmentID associates the variable with an environment (optional scope).
	EnvironmentID string `json:"environment_id,omitempty" db:"environment_id"`

	// Name is the variable name used for interpolation (e.g., {{api_token}}).
	Name string `json:"name" db:"name"`

	// Value is the variable value. For secrets, this should not be exposed in logs.
	Value string `json:"value" db:"value"`

	// Type indicates the variable type (e.g., "text", "expression", "secret").
	Type string `json:"type" db:"type"`

	// Description provides human-readable description of the variable.
	Description string `json:"description" db:"description"`

	// Secret designates whether this is a sensitive value.
	Secret bool `json:"secret" db:"secret"`

	// Scope indicates the variable's scope/inheritance point.
	Scope string `json:"scope" db:"scope"`

	// CreatedAt tracks when the variable was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt tracks when the variable was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ErrNotFound is returned when a variable is not found.
var ErrNotFound = variableError("not found")

type variableError string

func (e variableError) Error() string {
	return string(e)
}

// NewVariable creates a new variable with the given name and value.
func NewVariable(name, value, typ string, secret bool) *Variable {
	return &Variable{
		Name:  name,
		Value: value,
		Type:  typ,
		Secret: secret,
		Scope: "request", // default scope
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// IsEmpty returns true if the variable has no name.
func (v *Variable) IsEmpty() bool {
	return v.Name == ""
}
package environment

import "time"

// Environment represents a collection of reusable variables for different deployment targets.
type Environment struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ErrNotFound is returned when an environment is not found.
var ErrNotFound = environmentError("not found")

type environmentError string

func (e environmentError) Error() string { return string(e) }

// NewEnvironment creates a new environment with the given name.
func NewEnvironment(name string) *Environment {
	now := time.Now().UTC()
	return &Environment{
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsEmpty returns true if the environment has no name.
func (e *Environment) IsEmpty() bool {
	return e.Name == ""
}

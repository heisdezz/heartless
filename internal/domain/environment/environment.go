package environment

import (
	"time"

	"heartless/internal/domain/id"
)

// Environment represents a named set of variables within a workspace.
type Environment struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// New creates an environment with a generated ID and timestamps.
func New(name string) *Environment {
	now := time.Now().UTC()
	return &Environment{
		ID:        id.New("env"),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewEnvironment is a compatibility alias for New.
func NewEnvironment(name string) *Environment { return New(name) }

var ErrNotFound = environmentError("not found")

type environmentError string

func (e environmentError) Error() string { return string(e) }

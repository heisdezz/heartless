package environment_service

import (
	"context"
	"time"

	"heartless/internal/domain/environment"
	env "heartless/internal/storage/sqlite/environment"
)

// EnvironmentService handles environment-related business logic.
type EnvironmentService interface {
	// Create creates a new environment with the given name.
	Create(ctx context.Context, name string) (*environment.Environment, error)
	// Get retrieves an environment by ID.
	Get(ctx context.Context, id string) (*environment.Environment, error)
	// Update updates an environment's name or description.
	Update(ctx context.Context, env *environment.Environment) error
	// Delete removes an environment by ID.
	Delete(ctx context.Context, id string) error
	// List returns all environments for a workspace.
	List(ctx context.Context, workspaceID string) ([]*environment.Environment, error)
	// SetActive sets the active environment.
	SetActive(ctx context.Context, id string) error
	// GetActive returns the currently active environment.
	GetActive(ctx context.Context) (*environment.Environment, error)
}

// environmentService is the concrete implementation of EnvironmentService using SQLite.
type environmentService struct {
	repo env.Repository
}

// NewEnvironmentService creates a new environment service with the given SQLite repository.
func NewEnvironmentService(repo env.Repository) EnvironmentService {
	return &environmentService{repo: repo}
}

// Create creates a new environment with the given name.
func (s *environmentService) Create(ctx context.Context, name string) (*environment.Environment, error) {
	env := environment.NewEnvironment(name)
	if err := s.repo.Save(ctx, env, ""); err != nil {
		return nil, err
	}
	return env, nil
}

// Get retrieves an environment by ID.
func (s *environmentService) Get(ctx context.Context, id string) (*environment.Environment, error) {
	return s.repo.Get(ctx, id)
}

// Update updates an environment's name or description.
func (s *environmentService) Update(ctx context.Context, env *environment.Environment) error {
	env.UpdatedAt = time.Now().UTC()
	if err := s.repo.Save(ctx, env, ""); err != nil {
		return err
	}
	return nil
}

// Delete removes an environment by ID.
func (s *environmentService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// List returns all environments for a workspace.
func (s *environmentService) List(ctx context.Context, workspaceID string) ([]*environment.Environment, error) {
	return s.repo.List(ctx, workspaceID)
}

// SetActive sets the active environment.
func (s *environmentService) SetActive(ctx context.Context, id string) error {
	// TODO: implement setting active environment
	return nil
}

// GetActive returns the currently active environment.
func (s *environmentService) GetActive(ctx context.Context) (*environment.Environment, error) {
	// TODO: implement getting active environment
	return nil, nil
}
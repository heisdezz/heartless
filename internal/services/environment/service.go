package environment

import (
	"context"

	domain "heartless/internal/domain/environment"
)

// Repository defines persistence operations for environments.
type Repository interface {
	Get(ctx context.Context, id string) (*domain.Environment, error)
	Save(ctx context.Context, env *domain.Environment, workspaceID string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, workspaceID string) ([]*domain.Environment, error)
}

// Service defines environment application workflows.
type Service interface {
	Create(ctx context.Context, workspaceID, name string) (*domain.Environment, error)
	Get(ctx context.Context, id string) (*domain.Environment, error)
	Update(ctx context.Context, env *domain.Environment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, workspaceID string) ([]*domain.Environment, error)
}

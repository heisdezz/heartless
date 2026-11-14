package variable

import (
	"context"

	domain "heartless/internal/domain/variable"
)

// Repository defines persistence operations for variables.
type Repository interface {
	Get(ctx context.Context, id string) (*domain.Variable, error)
	Save(ctx context.Context, id, workspaceID, environmentID string, v *domain.Variable) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, workspaceID, environmentID string) ([]*domain.Variable, error)
}

// Service defines variable application workflows.
type Service interface {
	Create(ctx context.Context, workspaceID, environmentID, name, value string) (*domain.Variable, error)
	Get(ctx context.Context, id string) (*domain.Variable, error)
	Update(ctx context.Context, v *domain.Variable) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, workspaceID, environmentID string) ([]*domain.Variable, error)
}

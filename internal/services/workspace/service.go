package workspace

import (
	"context"

	domain "heartless/internal/domain/workspace"
)

// Repository defines persistence operations for workspaces.
type Repository interface {
	Get(ctx context.Context, id string) (*domain.Workspace, error)
	Save(ctx context.Context, ws *domain.Workspace) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, workspaceID string) ([]*domain.Workspace, error)
	Exists(ctx context.Context, name string) (bool, error)
	GetActiveEnvironmentID(ctx context.Context, workspaceID string) (string, error)
	SetActiveEnvironmentID(ctx context.Context, workspaceID, environmentID string) error
}

// Service defines workspace application workflows.
type Service interface {
	Create(ctx context.Context, name string) (*domain.Workspace, error)
	Get(ctx context.Context, id string) (*domain.Workspace, error)
	Update(ctx context.Context, ws *domain.Workspace) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.Workspace, error)
	GetActiveEnvironmentID(ctx context.Context, workspaceID string) (string, error)
	SetActiveEnvironmentID(ctx context.Context, workspaceID, environmentID string) error
}

package history

import (
	"context"

	domain "heartless/internal/domain/history"
)

// Repository defines persistence operations for execution history.
type Repository interface {
	Save(ctx context.Context, entry *domain.Entry) error
	Get(ctx context.Context, id string) (*domain.Entry, error)
	List(ctx context.Context, workspaceID string, limit int) ([]*domain.Entry, error)
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context, workspaceID string) error
}

// Service defines history query and retention workflows.
type Service interface {
	Record(ctx context.Context, entry *domain.Entry) error
	Get(ctx context.Context, id string) (*domain.Entry, error)
	List(ctx context.Context, workspaceID string, limit int) ([]*domain.Entry, error)
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context, workspaceID string) error
}

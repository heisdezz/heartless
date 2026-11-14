package request

import (
	"context"

	domain "heartless/internal/domain/request"
	"heartless/internal/domain/response"
)

// Transport executes a request and returns a structured result.
type Transport interface {
	Execute(ctx context.Context, req *domain.Request) (*response.ExecutionResult, error)
}

// Service defines request authoring and execution workflows.
type Service interface {
	Save(ctx context.Context, req *domain.Request) error
	Get(ctx context.Context, id string) (*domain.Request, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, workspaceID string) ([]*domain.Request, error)
	Execute(ctx context.Context, req *domain.Request) (*response.ExecutionResult, error)
	Cancel(ctx context.Context, executionID string) error
}

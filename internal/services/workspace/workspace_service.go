package workspace_service

import (
	"context"
	"time"

	"heartless/internal/domain/workspace"
	ws "heartless/internal/storage/sqlite/workspace"
)

// WorkspaceService handles workspace-related business logic.
// It depends on the repository interface, keeping domain logic separate from persistence.
type WorkspaceService interface {
	// Create creates a new workspace with the given name.
	Create(ctx context.Context, name string) (*workspace.Workspace, error)
	// Get retrieves a workspace by ID.
	Get(ctx context.Context, id string) (*workspace.Workspace, error)
	// Update updates a workspace's name or description.
	Update(ctx context.Context, ws *workspace.Workspace) error
	// Delete removes a workspace by ID.
	Delete(ctx context.Context, id string) error
	// List returns all workspaces, optionally filtered by parent workspace ID.
	List(ctx context.Context, workspaceID string) ([]*workspace.Workspace, error)
	// SetActive sets the active workspace.
	SetActive(ctx context.Context, id string) error
	// GetActive returns the currently active workspace.
	GetActive(ctx context.Context) (*workspace.Workspace, error)
}

// workspaceService is the concrete implementation of WorkspaceService using SQLite.
type workspaceService struct {
	repo ws.Repository
}

// NewWorkspaceService creates a new workspace service with the given SQLite repository.
func NewWorkspaceService(repo ws.Repository) WorkspaceService {
	return &workspaceService{repo: repo}
}

// Create creates a new workspace with the given name.
func (s *workspaceService) Create(ctx context.Context, name string) (*workspace.Workspace, error) {
	ws := workspace.NewWorkspace(name)
	if err := s.repo.Save(ctx, ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// Get retrieves a workspace by ID.
func (s *workspaceService) Get(ctx context.Context, id string) (*workspace.Workspace, error) {
	return s.repo.Get(ctx, id)
}

// Update updates a workspace's name or description.
func (s *workspaceService) Update(ctx context.Context, ws *workspace.Workspace) error {
	ws.UpdatedAt = time.Now().UTC()
	if err := s.repo.Save(ctx, ws); err != nil {
		return err
	}
	return nil
}

// Delete removes a workspace by ID.
func (s *workspaceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// List returns all workspaces, optionally filtered by parent workspace ID.
func (s *workspaceService) List(ctx context.Context, workspaceID string) ([]*workspace.Workspace, error) {
	return s.repo.List(ctx, workspaceID)
}

// SetActive sets the active workspace.
func (s *workspaceService) SetActive(ctx context.Context, id string) error {
	// TODO: implement setting active workspace
	return nil
}

// GetActive returns the currently active workspace.
func (s *workspaceService) GetActive(ctx context.Context) (*workspace.Workspace, error) {
	// TODO: implement getting active workspace
	return nil, nil
}
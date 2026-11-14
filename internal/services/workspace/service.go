package workspace

import (
	"context"
	"database/sql"

	col "heartless/internal/domain/collection"
	domain "heartless/internal/domain/workspace"
	col_repo "heartless/internal/storage/sqlite/collection"
	ws "heartless/internal/storage/sqlite/workspace"
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
	// Collection methods
	GetCollections(ctx context.Context, workspaceID string) ([]*col.Collection, error)
	GetCollection(ctx context.Context, id string) (*col.Collection, error)
	CreateCollection(ctx context.Context, workspaceID, name string) (*col.Collection, error)
	UpdateCollection(ctx context.Context, col *col.Collection) error
	DeleteCollection(ctx context.Context, id string) error
	// Workspace settings
	GetWorkspaceSettings(ctx context.Context, workspaceID string) (*domain.Workspace, error)
	UpdateWorkspaceSettings(ctx context.Context, ws *domain.Workspace) error
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
	// Collection methods
	GetCollections(ctx context.Context, workspaceID string) ([]*col.Collection, error)
	GetCollection(ctx context.Context, id string) (*col.Collection, error)
	CreateCollection(ctx context.Context, workspaceID, name string) (*col.Collection, error)
	UpdateCollection(ctx context.Context, col *col.Collection) error
	DeleteCollection(ctx context.Context, id string) error
	// Workspace settings
	GetWorkspaceSettings(ctx context.Context, workspaceID string) (*domain.Workspace, error)
	UpdateWorkspaceSettings(ctx context.Context, ws *domain.Workspace) error
}

// NewService creates a new workspace service with the given SQLite database.
func NewService(db *sql.DB) *service {
	// Ensure migration runs
	// repo.MigrateDB(repoSQL) // Would be called during app init
	repo := ws.New(db)
	collectionRepo := col_repo.New(db)
	return &service{repo: repo, collectionRepo: collectionRepo}
}

// service is the concrete implementation of Service using SQLite.
type service struct {
	repo           *ws.Repository
	collectionRepo *col_repo.Repository
}

// GetCollections returns all collections for a workspace.
func (s *service) GetCollections(ctx context.Context, workspaceID string) ([]*col.Collection, error) {
	return s.collectionRepo.ListCollections(ctx, workspaceID)
}

// GetCollection returns a collection by ID.
func (s *service) GetCollection(ctx context.Context, id string) (*col.Collection, error) {
	return s.collectionRepo.GetCollection(ctx, id)
}

// CreateCollection creates a new collection in a workspace.
func (s *service) CreateCollection(ctx context.Context, workspaceID, name string) (*col.Collection, error) {
	col := col.New(workspaceID, name)
	if err := s.collectionRepo.SaveCollection(ctx, col); err != nil {
		return nil, err
	}
	return col, nil
}

// UpdateCollection updates a collection's name or sort order.
func (s *service) UpdateCollection(ctx context.Context, col *col.Collection) error {
	if err := s.collectionRepo.SaveCollection(ctx, col); err != nil {
		return err
	}
	return nil
}

// DeleteCollection removes a collection by ID.
func (s *service) DeleteCollection(ctx context.Context, id string) error {
	return s.collectionRepo.DeleteCollection(ctx, id)
}

// GetWorkspaceSettings returns the workspace settings for a given workspace.
func (s *service) GetWorkspaceSettings(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	return s.repo.Get(ctx, workspaceID)
}

// UpdateWorkspaceSettings updates the workspace settings.
func (s *service) UpdateWorkspaceSettings(ctx context.Context, ws *domain.Workspace) error {
	return s.repo.Save(ctx, ws)
}

package collection

import (
	"context"

	domain "heartless/internal/domain/collection"
	"heartless/internal/domain/request"
)

// Repository defines persistence operations for collections, folders, and saved requests.
type Repository interface {
	GetCollection(ctx context.Context, id string) (*domain.Collection, error)
	SaveCollection(ctx context.Context, col *domain.Collection) error
	DeleteCollection(ctx context.Context, id string) error
	ListCollections(ctx context.Context, workspaceID string) ([]*domain.Collection, error)

	GetFolder(ctx context.Context, id string) (*domain.Folder, error)
	SaveFolder(ctx context.Context, folder *domain.Folder) error
	DeleteFolder(ctx context.Context, id string) error
	ListFolders(ctx context.Context, collectionID string) ([]*domain.Folder, error)

	GetRequest(ctx context.Context, id string) (*request.Request, error)
	SaveRequest(ctx context.Context, req *request.Request) error
	DeleteRequest(ctx context.Context, id string) error
	ListRequests(ctx context.Context, workspaceID, collectionID, folderID string) ([]*request.Request, error)
}

// Service defines collection/folder/request organization workflows.
type Service interface {
	CreateCollection(ctx context.Context, workspaceID, name string) (*domain.Collection, error)
	ListCollections(ctx context.Context, workspaceID string) ([]*domain.Collection, error)
	CreateFolder(ctx context.Context, collectionID, parentID, name string) (*domain.Folder, error)
	ListFolders(ctx context.Context, collectionID string) ([]*domain.Folder, error)
	SaveRequest(ctx context.Context, req *request.Request) error
	GetRequest(ctx context.Context, id string) (*request.Request, error)
	DeleteRequest(ctx context.Context, id string) error
	ListRequests(ctx context.Context, workspaceID, collectionID, folderID string) ([]*request.Request, error)
}

package collection

import (
	"time"

	"heartless/internal/domain/id"
)

// Collection groups folders and requests within a workspace.
type Collection struct {
	ID          string    `json:"id" db:"id"`
	WorkspaceID string    `json:"workspace_id" db:"workspace_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	SortOrder   int       `json:"sort_order,omitempty" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// New creates a collection with a generated ID and timestamps.
func New(workspaceID, name string) *Collection {
	now := time.Now().UTC()
	return &Collection{
		ID:          id.New("col"),
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Folder organizes requests hierarchically inside a collection.
type Folder struct {
	ID           string    `json:"id" db:"id"`
	CollectionID string    `json:"collection_id" db:"collection_id"`
	ParentID     string    `json:"parent_id,omitempty" db:"parent_id"`
	Name         string    `json:"name" db:"name"`
	Description  string    `json:"description,omitempty" db:"description"`
	SortOrder    int       `json:"sort_order,omitempty" db:"sort_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// NewFolder creates a folder with a generated ID and timestamps.
func NewFolder(collectionID, parentID, name string) *Folder {
	now := time.Now().UTC()
	return &Folder{
		ID:           id.New("fld"),
		CollectionID: collectionID,
		ParentID:     parentID,
		Name:         name,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ErrCollectionNotFound is returned when no collection has the requested ID.
var ErrCollectionNotFound = collectionError("collection not found")

// ErrFolderNotFound is returned when no folder has the requested ID.
var ErrFolderNotFound = collectionError("folder not found")

// ErrRequestNotFound is returned when no saved request has the requested ID.
var ErrRequestNotFound = collectionError("request not found")

type collectionError string

func (e collectionError) Error() string { return string(e) }

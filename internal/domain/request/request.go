package request

import (
	"time"

	"heartless/internal/domain/id"
)

// Kind identifies the protocol/type of a request.
type Kind string

const (
	KindHTTP      Kind = "http"
	KindGraphQL   Kind = "graphql"
	KindGRPC      Kind = "grpc"
	KindWebSocket Kind = "websocket"
)

// Request is a saved API request definition (independent of execution state).
type Request struct {
	ID           string            `json:"id" db:"id"`
	WorkspaceID  string            `json:"workspace_id" db:"workspace_id"`
	CollectionID string            `json:"collection_id,omitempty" db:"collection_id"`
	FolderID     string            `json:"folder_id,omitempty" db:"folder_id"`
	Name         string            `json:"name" db:"name"`
	Description  string            `json:"description,omitempty" db:"description"`
	Kind         Kind              `json:"kind" db:"kind"`
	Method       string            `json:"method,omitempty" db:"method"`
	URL          string            `json:"url,omitempty" db:"url"`
	Headers      map[string]string `json:"headers,omitempty" db:"-"`
	Params       map[string]string `json:"params,omitempty" db:"-"`
	Body         string            `json:"body,omitempty" db:"body"`
	BodyType     string            `json:"body_type,omitempty" db:"body_type"`
	SortOrder    int               `json:"sort_order,omitempty" db:"sort_order"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// New creates a request with a generated ID and timestamps.
func New(workspaceID, name string, kind Kind) *Request {
	now := time.Now().UTC()
	return &Request{
		ID:          id.New("req"),
		WorkspaceID: workspaceID,
		Name:        name,
		Kind:        kind,
		Headers:     map[string]string{},
		Params:      map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

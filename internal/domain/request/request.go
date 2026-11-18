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
	// Form holds the body fields of the form and multipart body types; it is
	// encoded into the request body at execution time.
	Form     map[string]string `json:"form,omitempty" db:"-"`
	Body     string            `json:"body,omitempty" db:"body"`
	BodyType string            `json:"body_type,omitempty" db:"body_type"`
	// Script is the JavaScript run after the response arrives.
	Script string `json:"script,omitempty" db:"script"`
	// Auth is the request's authentication, applied at execution time.
	Auth      Auth      `json:"auth,omitempty" db:"-"`
	SortOrder int       `json:"sort_order,omitempty" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AuthType identifies how a request authenticates.
type AuthType string

// Auth types, matching the names the editor offers.
const (
	AuthInherit AuthType = "inherit"
	AuthNone    AuthType = "none"
	AuthAPIKey  AuthType = "api_key"
	AuthAWS     AuthType = "aws"
	AuthBasic   AuthType = "basic"
	AuthBearer  AuthType = "bearer"
	AuthDigest  AuthType = "digest"
	AuthJWT     AuthType = "jwt"
	AuthOAuth1  AuthType = "oauth1"
	AuthOAuth2  AuthType = "oauth2"
	AuthNTLM    AuthType = "ntlm"
)

// Auth holds a request's authentication settings. Only the fields its Type
// needs are used.
type Auth struct {
	Type AuthType `json:"type"`
	// Token is the bearer token or JWT.
	Token string `json:"token,omitempty"`
	// Username and Password are used by Basic and Digest auth.
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Key, Header and Prefix are used by API key auth.
	Key    string `json:"key,omitempty"`
	Header string `json:"header,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

// IsZero reports whether the auth adds nothing, so it need not be stored.
func (a Auth) IsZero() bool {
	return a == Auth{}
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

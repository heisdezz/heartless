package events

import (
	"context"
	"time"

	"heartless/internal/domain/environment"
	"heartless/internal/domain/history"
	"heartless/internal/domain/request"
	"heartless/internal/domain/response"
	"heartless/internal/domain/variable"
	"heartless/internal/domain/workspace"
)

// Type identifies a typed application event.
type Type string

const (
	WorkspaceCreated      Type = "workspace.created"
	WorkspaceUpdated      Type = "workspace.updated"
	WorkspaceDeleted      Type = "workspace.deleted"
	WorkspaceActivated    Type = "workspace.activated"
	EnvironmentCreated    Type = "environment.created"
	EnvironmentUpdated    Type = "environment.updated"
	EnvironmentDeleted    Type = "environment.deleted"
	EnvironmentActivated  Type = "environment.activated"
	VariableCreated       Type = "variable.created"
	VariableUpdated       Type = "variable.updated"
	VariableDeleted       Type = "variable.deleted"
	RequestCreated        Type = "request.created"
	RequestUpdated        Type = "request.updated"
	RequestDeleted        Type = "request.deleted"
	RequestStarted        Type = "request.started"
	RequestCompleted      Type = "request.completed"
	RequestFailed         Type = "request.failed"
	RequestCancelled      Type = "request.cancelled"
	HistoryRecorded       Type = "history.recorded"
	WebSocketConnected    Type = "websocket.connected"
	WebSocketDisconnected Type = "websocket.disconnected"
	GRPCConnected         Type = "grpc.connected"
	GRPCDisconnected      Type = "grpc.disconnected"
)

// Event is the common envelope for application events.
type Event struct {
	Type        Type      `json:"type"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	EntityID    string    `json:"entity_id,omitempty"`
	At          time.Time `json:"at"`
	Error       string    `json:"error,omitempty"`
	Payload     any       `json:"payload,omitempty"`
}

// New creates an event with the current UTC timestamp.
func New(t Type, workspaceID, entityID string, payload any) Event {
	return Event{
		Type:        t,
		WorkspaceID: workspaceID,
		EntityID:    entityID,
		At:          time.Now().UTC(),
		Payload:     payload,
	}
}

// WorkspacePayload carries workspace state changes.
type WorkspacePayload struct {
	Workspace *workspace.Workspace `json:"workspace,omitempty"`
}

// EnvironmentPayload carries environment state changes.
type EnvironmentPayload struct {
	Environment *environment.Environment `json:"environment,omitempty"`
	Active      bool                     `json:"active,omitempty"`
}

// VariablePayload carries variable state changes.
type VariablePayload struct {
	Variable *variable.Variable `json:"variable,omitempty"`
}

// RequestPayload carries request definition state changes.
type RequestPayload struct {
	Request *request.Request `json:"request,omitempty"`
}

// ExecutionPayload carries execution lifecycle/result data.
type ExecutionPayload struct {
	RequestID string                    `json:"request_id,omitempty"`
	Result    *response.ExecutionResult `json:"result,omitempty"`
	Error     string                    `json:"error,omitempty"`
}

// HistoryPayload carries history persistence events.
type HistoryPayload struct {
	Entry *history.Entry `json:"entry,omitempty"`
}

// Handler processes an event.
type Handler func(context.Context, Event)

// Publisher publishes typed application events.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Subscriber subscribes handlers to event types.
type Subscriber interface {
	Subscribe(eventType Type, handler Handler) (unsubscribe func())
}

// Bus combines publishing and subscription for in-process events.
type Bus interface {
	Publisher
	Subscriber
}

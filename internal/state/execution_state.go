package state

import "time"

// ExecutionStatus represents transient status of a request execution or connection.
type ExecutionStatus string

const (
	ExecutionIdle      ExecutionStatus = "idle"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionSucceeded ExecutionStatus = "succeeded"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCanceled  ExecutionStatus = "canceled"
)

// ExecutionState tracks transient state for an in-flight or recent execution.
type ExecutionState struct {
	ID         string
	RequestID  string
	Status     ExecutionStatus
	StartedAt  time.Time
	EndedAt    time.Time
	Error      string
	Cancelable bool
}

// ConnectionStatus represents transient protocol connection state.
type ConnectionStatus string

const (
	ConnectionDisconnected ConnectionStatus = "disconnected"
	ConnectionConnecting   ConnectionStatus = "connecting"
	ConnectionConnected    ConnectionStatus = "connected"
	ConnectionError        ConnectionStatus = "error"
)

// ConnectionState tracks transient state for persistent connections (WS/gRPC streams).
type ConnectionState struct {
	ID        string
	RequestID string
	Status    ConnectionStatus
	Error     string
	UpdatedAt time.Time
}

// TransientState aggregates transient execution and connection state.
// It must not contain persisted domain data.
type TransientState struct {
	Executions  map[string]*ExecutionState
	Connections map[string]*ConnectionState
	LastError   string
}

// NewTransientState returns initialized transient state.
func NewTransientState() TransientState {
	return TransientState{
		Executions:  map[string]*ExecutionState{},
		Connections: map[string]*ConnectionState{},
	}
}

// SetExecution stores or updates transient execution state.
func (s *TransientState) SetExecution(exec *ExecutionState) {
	if s.Executions == nil {
		s.Executions = map[string]*ExecutionState{}
	}
	if exec == nil || exec.ID == "" {
		return
	}
	s.Executions[exec.ID] = exec
}

// SetConnection stores or updates transient connection state.
func (s *TransientState) SetConnection(conn *ConnectionState) {
	if s.Connections == nil {
		s.Connections = map[string]*ConnectionState{}
	}
	if conn == nil || conn.ID == "" {
		return
	}
	s.Connections[conn.ID] = conn
}

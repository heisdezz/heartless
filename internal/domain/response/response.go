package response

import "time"

// Response captures the result of executing a request.
type Response struct {
	ID           string            `json:"id"`
	RequestID    string            `json:"request_id,omitempty"`
	StatusCode   int               `json:"status_code"`
	StatusText   string            `json:"status_text,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         []byte            `json:"body,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Size         int64             `json:"size"`
	Duration     time.Duration     `json:"duration"`
	EffectiveURL string            `json:"effective_url,omitempty"`
	Error        string            `json:"error,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// ExecutionResult represents a completed request execution.
type ExecutionResult struct {
	RequestID string        `json:"request_id,omitempty"`
	Response  *Response     `json:"response,omitempty"`
	Duration  time.Duration `json:"duration"`
	Error     error         `json:"-"`
	ErrorText string        `json:"error,omitempty"`
	StartedAt time.Time     `json:"started_at,omitempty"`
	EndedAt   time.Time     `json:"ended_at,omitempty"`
}

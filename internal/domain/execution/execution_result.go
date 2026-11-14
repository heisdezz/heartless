package execution

import (
	"net/http"
	"time"
)

// ExecutionResult represents the result of executing a request.
// It is returned by the execution engine rather than modifying the original request.
type ExecutionResult struct {
	// RequestID is the ID of the executed request.
	RequestID string `json:"request_id" db:"request_id"`

	// Response contains the HTTP response captured from execution.
	Response *http.Response `json:"response,omitempty" db:"response"`

	// Duration is the total execution duration.
	Duration time.Duration `json:"duration" db:"duration"`

	// Error contains any error that occurred during execution.
	Error error `json:"error,omitempty" db:"error"`

	// DNSLookupDuration is the DNS lookup duration.
	DNSLookupDuration time.Duration `json:"dns_lookup_duration,omitempty" db:"dns_lookup_duration"`

	// ConnectionDuration is the connection duration.
	ConnectionDuration time.Duration `json:"connection_duration,omitempty" db:"connection_duration"`

	// TLSHandshakeDuration is the TLS handshake duration.
	TLSHandshakeDuration time.Duration `json:"tls_handshake_duration,omitempty" db:"tls_handshake_duration"`

	// TimeToFirstByte is the time to first byte.
	TimeToFirstByte time.Duration `json:"time_to_first_byte,omitempty" db:"time_to_first_byte"`

	// DownloadDuration is the download duration.
	DownloadDuration time.Duration `json:"download_duration,omitempty" db:"download_duration"`

	// EffectiveURL is the effective URL after following redirects.
	EffectiveURL string `json:"effective_url" db:"effective_url"`

	// RedirectInformation contains redirect details.
	RedirectInformation *RedirectInfo `json:"redirect_info,omitempty" db:"redirect_info"`
}

// RedirectInfo contains redirect-related information.
type RedirectInfo struct {
	// MaxRedirects is the maximum number of redirects followed.
	MaxRedirects int `json:"max_redirects" db:"max_redirects"`

	// FollowedRedirects is the number of redirects actually followed.
	FollowedRedirects int `json:"followed_redirects" db:"followed_redirects"`

	// FinalURL is the final URL after all redirects.
	FinalURL string `json:"final_url" db:"final_url"`
}

// NewExecutionResult creates a new ExecutionResult with the given request ID.
func NewExecutionResult(requestID string) *ExecutionResult {
	return &ExecutionResult{
		RequestID: requestID,
	}
}
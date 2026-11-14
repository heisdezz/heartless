package response

import "time"

// Timing captures HTTP request/response timing details.
type Timing struct {
	DNSLookup    time.Duration `json:"dns_lookup,omitempty"`
	Connect      time.Duration `json:"connect,omitempty"`
	TLSHandshake time.Duration `json:"tls_handshake,omitempty"`
	FirstByte    time.Duration `json:"first_byte,omitempty"`
	Download     time.Duration `json:"download,omitempty"`
	Total        time.Duration `json:"total,omitempty"`
}

// HTTPMetadata holds additional HTTP response metadata.
type HTTPMetadata struct {
	EffectiveURL string            `json:"effective_url,omitempty"`
	Redirects    []string          `json:"redirects,omitempty"`
	RequestID    string            `json:"request_id,omitempty"`
	Timing       Timing            `json:"timing,omitempty"`
	HeadersSize  int64             `json:"headers_size,omitempty"`
	BodySize     int64             `json:"body_size,omitempty"`
	Trailers     map[string]string `json:"trailers,omitempty"`
}

// NewHTTPResponse creates a response with HTTP defaults populated.
func NewHTTPResponse(requestID string, status int, statusText string) *Response {
	return &Response{
		RequestID:  requestID,
		StatusCode: status,
		StatusText: statusText,
		Headers:    map[string]string{},
		CreatedAt:  time.Now().UTC(),
	}
}

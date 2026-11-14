package response

import "testing"

func TestNewHTTPResponse(t *testing.T) {
	r := NewHTTPResponse("req1", 200, "OK")
	if r.RequestID != "req1" || r.StatusCode != 200 || r.StatusText != "OK" {
		t.Fatalf("unexpected response: %+v", r)
	}
	if r.Headers == nil || r.CreatedAt.IsZero() {
		t.Fatalf("expected headers and timestamp: %+v", r)
	}
}

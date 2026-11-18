package screens

import (
	"testing"

	domainres "heartless/internal/domain/response"
)

// TestResponseHeadersIncludeURL checks the Headers tab lists the URL the
// request was actually sent to, after redirects, alongside the response
// headers, and that cookies still come from the real headers.
func TestResponseHeadersIncludeURL(t *testing.T) {
	st := shellState{requestStates: map[string]*requestState{
		"req-1": {response: &domainres.ExecutionResult{
			Response: &domainres.Response{
				StatusCode:   200,
				Headers:      map[string]string{"Content-Type": "text/plain", "Set-Cookie": "sid=1"},
				EffectiveURL: "https://api.test/final",
			},
		}},
	}}

	resp, _, _, _, _ := st.responseView("req-1")
	if resp == nil {
		t.Fatal("no response snapshot")
	}
	if resp.Headers["URL"] != "https://api.test/final" {
		t.Fatalf("headers = %v, want URL entry", resp.Headers)
	}
	if resp.Headers["Content-Type"] != "text/plain" {
		t.Fatalf("headers = %v, want real headers kept", resp.Headers)
	}
	if len(resp.Cookies) != 1 || resp.Cookies[0] != "sid=1" {
		t.Fatalf("cookies = %v, want sid=1", resp.Cookies)
	}
}

package screens

import (
	"context"
	"testing"

	domainhist "heartless/internal/domain/history"
	domainres "heartless/internal/domain/response"
)

// TestRequestResponseTintState checks that requests with a persisted response
// are tracked for the sidebar name tint, and that a send marks them immediately.
func TestRequestResponseTintState(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	withResponse := domainhist.New("ws-1")
	withResponse.RequestID = "req-with-response"
	withResponse.StatusCode = 200
	withResponse.Response = &domainres.Response{
		StatusCode:  200,
		StatusText:  "OK",
		Body:        []byte(`{"ok":true}`),
		ContentType: "application/json",
		Size:        int64(len(`{"ok":true}`)),
	}
	if err := services.History.Record(ctx, withResponse); err != nil {
		t.Fatalf("record history: %v", err)
	}

	st := shellState{workspaceID: "ws-1"}
	st.refreshRequestsWithResponses(services)
	if !st.requestHasResponse("req-with-response") {
		t.Fatal("request with saved response was not marked")
	}
	if st.requestHasResponse("req-without-response") {
		t.Fatal("request without a saved response was marked")
	}

	// A send marks a request without needing another history scan.
	st.markRequestHasResponse("req-without-response")
	if !st.requestHasResponse("req-without-response") {
		t.Fatal("mark after send did not tint the request")
	}
}

package screens

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	domainreq "heartless/internal/domain/request"
	domainvar "heartless/internal/domain/variable"
)

// TestSendPersistsResponseToHistory runs a request against a local server and
// checks the response, with its body and headers, is stored in the database.
func TestSendPersistsResponseToHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":7}`))
	}))
	defer srv.Close()

	services := newTestServices(t)
	base := domainvar.NewVariable("baseUrl", srv.URL, "text", false)
	base.ID = "v1"
	base.WorkspaceID = "ws-1"

	st := shellState{
		workspaceID: "ws-1",
		variables:   []domainvar.Variable{*base},
	}
	req := domainreq.New("ws-1", "Create", domainreq.KindHTTP)
	req.ID = "req-hist"
	req.Method = "GET"
	req.URL = "{{baseUrl}}/items"
	st.requestStates = map[string]*requestState{
		req.ID: {Method: "GET", URL: "{{baseUrl}}/items", Params: []*kv{}, Headers: []*kv{}},
	}

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	entries, err := services.History.List(context.Background(), "ws-1", 10)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.RequestID != req.ID || e.StatusCode != http.StatusCreated {
		t.Fatalf("entry = %+v", e)
	}
	if e.URL != srv.URL+"/items" {
		t.Fatalf("url = %q, want resolved %q", e.URL, srv.URL+"/items")
	}
	if e.Response == nil {
		t.Fatalf("response not stored")
	}
	if got := string(e.Response.Body); got != `{"id":7}` {
		t.Fatalf("body = %q", got)
	}
	if e.Response.Headers["X-Test"] != "yes" || e.Response.ContentType != "application/json" {
		t.Fatalf("headers = %+v contentType = %q", e.Response.Headers, e.Response.ContentType)
	}
	if e.Response.StatusText == "" {
		t.Fatalf("status text empty")
	}
}

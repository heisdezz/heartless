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
	if e.Response.ID == "" || e.Response.RequestID != req.ID {
		t.Fatalf("response identity not stored: id=%q request_id=%q", e.Response.ID, e.Response.RequestID)
	}
	if e.Response.EffectiveURL != srv.URL+"/items" {
		t.Fatalf("effective url = %q", e.Response.EffectiveURL)
	}
	if e.Response.Duration <= 0 {
		t.Fatalf("duration = %v", e.Response.Duration)
	}
	if e.Response.Size != int64(len(`{"id":7}`)) {
		t.Fatalf("size = %d", e.Response.Size)
	}
}

// TestLoadResponseFromHistory checks that reopening a request seeds its response
// pane with the most recent result persisted in history.
func TestLoadResponseFromHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	services := newTestServices(t)
	req := domainreq.New("ws-1", "Reload", domainreq.KindHTTP)
	req.ID = "req-reload"
	req.Method = "GET"
	req.URL = srv.URL

	// Record one execution so there is a persisted response to find.
	sender := shellState{
		workspaceID: "ws-1",
		requestStates: map[string]*requestState{
			req.ID: {Method: "GET", URL: req.URL, Params: []*kv{}, Headers: []*kv{}},
		},
	}
	done := make(chan struct{})
	sender.send(services, *req, func() { close(done) })
	<-done

	// A fresh state (e.g. after a restart) has no in-memory response.
	rs := &requestState{Method: "GET", URL: req.URL}
	st := shellState{}
	st.loadResponseFromHistory(services, req.ID, rs)
	if rs.response == nil || rs.response.Response == nil {
		t.Fatalf("history response not loaded: %+v", rs.response)
	}
	if got := string(rs.response.Response.Body); got != `{"ok":true}` {
		t.Fatalf("loaded body = %q", got)
	}
	if rs.response.Response.StatusCode != http.StatusOK {
		t.Fatalf("loaded status = %d", rs.response.Response.StatusCode)
	}

	// It loads only once, so a later send's fresh response is not overwritten.
	rs.response.Response.Body = []byte("newer")
	st.loadResponseFromHistory(services, req.ID, rs)
	if got := string(rs.response.Response.Body); got != "newer" {
		t.Fatalf("history load overwrote current response: %q", got)
	}
}

package screens

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
	domainvar "heartless/internal/domain/variable"
)

// TestBearerAuthHeaderApplied checks a bearer token reaches the server as an
// Authorization header, with {{var}} resolved from the environment.
func TestBearerAuthHeaderApplied(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	services := newTestServices(t)
	tok := domainvar.NewVariable("token", "s3cret", "text", false)
	tok.ID = "v1"
	tok.WorkspaceID = "ws-1"

	st := shellState{workspaceID: "ws-1", variables: []domainvar.Variable{*tok}}
	req := domainreq.New("ws-1", "Authed", domainreq.KindHTTP)
	req.ID = "req-auth"
	req.Method = "GET"
	req.URL = srv.URL + "/x"
	req.Auth = domainreq.Auth{Type: domainreq.AuthBearer, Token: "{{token}}"}
	st.requestStates = map[string]*requestState{
		req.ID: {
			Method: "GET", URL: req.URL,
			Params: []*kv{}, Headers: []*kv{},
			Auth: domainreq.Auth{Type: domainreq.AuthBearer, Token: "{{token}}"},
		},
	}

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	if gotAuth != "Bearer s3cret" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Bearer s3cret")
	}
}

// TestAuthPersists checks the auth settings round-trip through the database.
func TestAuthPersists(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	col := domaincol.New("ws-1", "api")
	if err := services.Collections.SaveCollection(ctx, col); err != nil {
		t.Fatalf("collection: %v", err)
	}
	req := domainreq.New("ws-1", "Authed", domainreq.KindHTTP)
	req.CollectionID = col.ID
	req.Auth = domainreq.Auth{Type: domainreq.AuthBearer, Token: "abc123"}
	if err := services.Collections.SaveRequest(ctx, req); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := services.Collections.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Auth.Type != domainreq.AuthBearer || got.Auth.Token != "abc123" {
		t.Fatalf("auth = %+v", got.Auth)
	}
}

// TestAuthMenuListsEveryType checks the menu offers the reference types.
func TestAuthMenuListsEveryType(t *testing.T) {
	items := authMenuItems(domainreq.AuthBearer)
	labels := map[string]bool{}
	for _, it := range items {
		if !it.Divider {
			labels[it.Label] = true
		}
	}
	for _, want := range []string{
		"API Key", "AWS Signature", "Basic Auth", "Bearer Token", "Digest Auth",
		"JWT Bearer", "OAuth 1.0", "OAuth 2.0", "NTLM Auth",
		"Inherit from Parent", "No Auth", "Copy from Folder",
	} {
		if !labels[want] {
			t.Errorf("menu missing %q", want)
		}
	}
	// The current type is checked.
	for _, it := range items {
		if it.Key == string(domainreq.AuthBearer) && it.Icon == nil {
			t.Errorf("current auth type not checked")
		}
	}
}

// TestRequestAndResponseSideBySide checks the response sits to the right of the
// request, not below it.
func TestRequestAndResponseSideBySide(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1"}
	req := *domainreq.New("ws-1", "Alpha", domainreq.KindHTTP)
	req.ID = "req-1"
	req.Method = "GET"
	req.URL = "https://api.test/x"
	st.requestStateFor(req)

	tt := ui.NewTester(func(c *ui.Context) {
		requestPane(c, theme.TokensOf(c), &st, services, req)
	}, 1400, 800)
	tt.Frame()

	sendRect, ok1 := tt.Find("Send")
	respRect, ok2 := tt.Find("Response")
	if !ok1 || !ok2 {
		t.Fatalf("missing panes: send=%v response=%v", ok1, ok2)
	}
	if respRect.X <= sendRect.X {
		t.Fatalf("response at x=%.0f is not right of request (send x=%.0f)", respRect.X, sendRect.X)
	}
}

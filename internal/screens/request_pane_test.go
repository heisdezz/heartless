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

// TestRequestStatesAreIsolated checks that two requests keep separate editors,
// responses and running flags.
func TestRequestStatesAreIsolated(t *testing.T) {
	st := shellState{}
	rsa := st.requestStateFor(domainreq.Request{ID: "a", URL: "https://a.test"})
	rsb := st.requestStateFor(domainreq.Request{ID: "b", URL: "https://b.test"})

	rsa.URL = "https://a.changed"
	if rsb.URL == "https://a.changed" {
		t.Fatalf("beta picked up alpha's URL")
	}

	mu.Lock()
	rsa.running = true
	mu.Unlock()
	if rsb.running {
		t.Fatalf("beta marked running when only alpha is")
	}
	if rsa.response != nil || rsb.response != nil {
		t.Fatalf("fresh states should have no response")
	}
}

// TestPlaceholderResolution checks {{var}} previews use environment values over
// workspace ones.
func TestPlaceholderResolution(t *testing.T) {
	envVar := domainvar.Variable{ID: "v1", Name: "baseUrl", Value: "https://dev.test", EnvironmentID: "env-dev"}
	wsVar := domainvar.Variable{ID: "v2", Name: "token", Value: "workspace-token"}
	st := shellState{
		workspaceID:   "ws-1",
		environmentID: "env-dev",
		variables:     []domainvar.Variable{wsVar, envVar},
	}
	if got := st.previewURL("{{baseUrl}}/users"); got != "https://dev.test/users" {
		t.Fatalf("preview = %q", got)
	}
	if got := st.previewURL("{{token}}"); got != "workspace-token" {
		t.Fatalf("workspace preview = %q", got)
	}
	if got := st.previewURL("{{missing}}/x"); got != "{{missing}}/x" {
		t.Fatalf("unknown placeholder = %q", got)
	}
}

// TestSendResolvesEnvironmentVariables runs a request whose URL uses {{baseUrl}}
// against a local server and checks the resolved URL was used.
func TestSendResolvesEnvironmentVariables(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	services := newTestServices(t)
	base := domainvar.NewVariable("baseUrl", srv.URL, "text", false)
	base.ID = "v1"
	base.WorkspaceID = "ws-1"

	st := shellState{
		workspaceID:   "ws-1",
		environmentID: "env-dev",
		variables:     []domainvar.Variable{*base},
	}
	req := domainreq.New("ws-1", "Users", domainreq.KindHTTP)
	req.ID = "req-x"
	req.Method = "GET"
	req.URL = "{{baseUrl}}/users"
	st.requestStates = map[string]*requestState{
		req.ID: {Method: "GET", URL: "{{baseUrl}}/users", Params: []*kv{}, Headers: []*kv{}},
	}

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	rs, ok := st.snapshot(req.ID)
	if !ok {
		t.Fatalf("no state for request")
	}
	if rs.error != "" {
		t.Fatalf("error: %s", rs.error)
	}
	if rs.response == nil || rs.response.Response == nil || rs.response.Response.StatusCode != 200 {
		t.Fatalf("response = %+v", rs.response)
	}
	if gotPath != "/users" {
		t.Fatalf("server path = %q, want /users (baseUrl resolved)", gotPath)
	}
}

// TestRequestPaneRenders checks selecting a request shows its editor.
func TestRequestPaneRenders(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	col := domaincol.New("ws-1", "api")
	_ = services.Collections.SaveCollection(ctx, col)
	req := domainreq.New("ws-1", "Alpha", domainreq.KindHTTP)
	req.CollectionID = col.ID
	req.Method = "POST"
	req.URL = "{{baseUrl}}/users"
	_ = services.Collections.SaveRequest(ctx, req)

	st := shellState{workspaceID: "ws-1", selectedRequestID: req.ID, router: ui.NewRouter("/")}
	st.load(ctx, services)

	tt := ui.NewTester(func(c *ui.Context) { workspaceView(c, theme.TokensOf(c), &st, services) }, 900, 700)
	for _, want := range []string{"Alpha", "Params", "Headers", "Body", "Send", "Response"} {
		if !tt.HasText(want) {
			t.Fatalf("pane missing %q, texts %q", want, tt.Texts())
		}
	}
}

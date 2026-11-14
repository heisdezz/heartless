package screens

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domainreq "heartless/internal/domain/request"
)

// TestRequestPaneHasAuthTab checks the request editor offers the Auth tab, shows
// the auth type menu there, and renders the credential fields for the type.
func TestRequestPaneHasAuthTab(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1"}
	req := *domainreq.New("ws-1", "Alpha", domainreq.KindHTTP)
	req.ID = "req-auth-ui"
	req.Method = "GET"
	req.URL = "https://api.test/x"
	rs := st.requestStateFor(req)
	rs.Tab = 3 // Auth
	rs.Auth = domainreq.Auth{Type: domainreq.AuthBearer, Token: "abc"}

	tt := ui.NewTester(func(c *ui.Context) {
		requestPane(c, theme.TokensOf(c), &st, services, req)
	}, 1200, 800)
	tt.Frame()

	for _, want := range []string{"Params", "Headers", "Body", "Auth"} {
		if !tt.HasText(want) {
			t.Fatalf("missing tab %q, texts %q", want, tt.Texts())
		}
	}
	// The auth bar shows the current scheme and its actions.
	if !tt.HasText("Bearer") || !tt.HasText("Settings") || !tt.HasText("Info") {
		t.Fatalf("auth bar missing, texts %q", tt.Texts())
	}
	// The bearer credential field is rendered.
	if !tt.HasText("Token") {
		t.Fatalf("token field missing, texts %q", tt.Texts())
	}
}

// TestRequestPaneHasBodyTypePicker checks the Body tab offers the body types.
func TestRequestPaneHasBodyTypePicker(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1"}
	req := *domainreq.New("ws-1", "Alpha", domainreq.KindHTTP)
	req.ID = "req-body-ui"
	req.Method = "POST"
	req.URL = "https://api.test/x"
	rs := st.requestStateFor(req)
	rs.Tab = 2 // Body

	tt := ui.NewTester(func(c *ui.Context) {
		requestPane(c, theme.TokensOf(c), &st, services, req)
	}, 1200, 800)
	tt.Frame()

	// The closed select shows the current type; the menu is opened by a click.
	if !tt.HasText(bodyTypes[len(bodyTypes)-1].label) {
		t.Fatalf("body type picker missing, texts %q", tt.Texts())
	}
	if err := tt.Click(bodyTypes[len(bodyTypes)-1].label); err != nil {
		t.Fatalf("open body type menu: %v", err)
	}
	tt.Frame()
	if !tt.HasText("JSON") || !tt.HasText("Url Encoded") {
		t.Fatalf("body type options missing, texts %q", tt.Texts())
	}
}

// TestAuthPersistsWithDraft checks saving the editor stores the auth settings.
func TestAuthPersistsWithDraft(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1"}
	req := *domainreq.New("ws-1", "Alpha", domainreq.KindHTTP)
	req.ID = "req-auth-save"
	req.Method = "GET"
	req.URL = "https://api.test/x"
	st.requestStates = map[string]*requestState{
		req.ID: {Method: "GET", URL: req.URL, Params: []*kv{}, Headers: []*kv{}, Form: []*kv{},
			Auth: domainreq.Auth{Type: domainreq.AuthBearer, Token: "tok"}},
	}
	// persistDraft resolves the request from loaded state.
	st.requests = []domainreq.Request{req}

	st.persistDraft(services, req.ID, st.requestStates[req.ID])

	got, ok := st.requestByID(req.ID)
	if !ok {
		t.Fatal("request not found")
	}
	if got.Auth.Type != domainreq.AuthBearer || got.Auth.Token != "tok" {
		t.Fatalf("persisted auth = %+v", got.Auth)
	}
}

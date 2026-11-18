package screens

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domainreq "heartless/internal/domain/request"
	domainws "heartless/internal/domain/workspace"
)

// TestWorkspaceSettingsAuthSaves checks the auth draft round-trips through the
// workspace repository and that the dirty flag tracks the draft.
func TestWorkspaceSettingsAuthSaves(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	st := shellState{workspaceID: "ws-1"}
	st.load(ctx, services)
	st.ensureWorkspaceSettings()

	if st.workspaceSettingsDirty() {
		t.Fatal("fresh draft marked dirty")
	}

	st.wsAuth = domainreq.Auth{Type: domainreq.AuthBearer, Token: "tok-1"}
	st.wsAutoInherit = true
	if !st.workspaceSettingsDirty() {
		t.Fatal("edited draft not marked dirty")
	}

	st.saveWorkspaceSettings(services)
	if st.err != "" {
		t.Fatalf("save: %v", st.err)
	}
	if st.workspaceSettingsDirty() {
		t.Fatal("draft still dirty after save")
	}

	got, err := services.Workspaces.Get(ctx, "ws-1")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if got.DefaultAuth.Type != domainreq.AuthBearer || got.DefaultAuth.Token != "tok-1" {
		t.Fatalf("default auth = %+v", got.DefaultAuth)
	}
	if !got.AutoInheritAuth {
		t.Fatal("auto-inherit not saved")
	}
}

// TestAutoInheritAppliesWorkspaceAuth checks a request with no auth of its own
// sends the workspace's default credentials when auto-inherit is on, and
// nothing when it is off.
func TestAutoInheritAppliesWorkspaceAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	services := newTestServices(t)

	send := func(autoInherit bool) {
		t.Helper()
		gotAuth = ""
		ws, _ := services.Workspaces.Get(context.Background(), "ws-1")
		ws.DefaultAuth = domainreq.Auth{Type: domainreq.AuthBearer, Token: "ws-token"}
		ws.AutoInheritAuth = autoInherit
		if err := services.Workspaces.Save(context.Background(), ws); err != nil {
			t.Fatalf("save workspace: %v", err)
		}

		st := shellState{workspaceID: "ws-1"}
		st.load(context.Background(), services)
		req := domainreq.New("ws-1", "Plain", domainreq.KindHTTP)
		req.ID = "req-inherit"
		req.Method = "GET"
		req.URL = srv.URL + "/x"
		st.requestStates = map[string]*requestState{
			req.ID: {Method: "GET", URL: req.URL, Params: []*kv{}, Headers: []*kv{}},
		}

		done := make(chan struct{})
		st.send(services, *req, func() { close(done) })
		<-done
	}

	send(true)
	if gotAuth != "Bearer ws-token" {
		t.Fatalf("Authorization = %q, want the workspace's bearer token", gotAuth)
	}

	send(false)
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want none with auto-inherit off", gotAuth)
	}
}

// TestInheritFollowsWorkspaceAuth checks a request set to Inherit takes the
// workspace's credentials regardless of the auto-inherit switch.
func TestInheritFollowsWorkspaceAuth(t *testing.T) {
	st := shellState{
		workspaceID: "ws-1",
		workspaces: []domainws.Workspace{
			{ID: "ws-1", DefaultAuth: domainreq.Auth{Type: domainreq.AuthBasic, Username: "u", Password: "p"}},
		},
	}
	got := st.resolveAuthInheritance(domainreq.Auth{Type: domainreq.AuthInherit})
	if got != (domainreq.Auth{Type: domainreq.AuthBasic, Username: "u", Password: "p"}) {
		t.Fatalf("inherited auth = %+v", got)
	}
}

// TestWorkspaceSettingsPageShowsSections checks the page renders its sections
// and the auth form.
func TestWorkspaceSettingsPageShowsSections(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/workspace-settings")}

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		workspaceSettingsView(c, theme.TokensOf(c), &st, services)
	}, 1000, 700)
	tt.Frame()

	for _, want := range []string{"Workspace Settings", "Auth", "Variables", "Environments", "Auto-inherit"} {
		if !tt.HasText(want) {
			t.Fatalf("page missing %q, texts %q", want, tt.Texts())
		}
	}
}

// TestWorkspaceSettingsEnvDraftIsolated checks switching workspaces reseeds the
// auth draft from the newly active one.
func TestWorkspaceSettingsEnvDraftIsolated(t *testing.T) {
	st := shellState{
		workspaceID: "ws-1",
		workspaces: []domainws.Workspace{
			{ID: "ws-1", DefaultAuth: domainreq.Auth{Type: domainreq.AuthBearer, Token: "one"}},
			{ID: "ws-2", DefaultAuth: domainreq.Auth{Type: domainreq.AuthBasic, Username: "two"}},
		},
	}
	st.ensureWorkspaceSettings()
	if st.wsAuth != (domainreq.Auth{Type: domainreq.AuthBearer, Token: "one"}) {
		t.Fatalf("first draft = %+v", st.wsAuth)
	}

	st.workspaceID = "ws-2"
	st.ensureWorkspaceSettings()
	if st.wsAuth != (domainreq.Auth{Type: domainreq.AuthBasic, Username: "two"}) {
		t.Fatalf("second draft = %+v", st.wsAuth)
	}
}

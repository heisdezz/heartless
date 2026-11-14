package screens

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"heartless/internal/app"
	domainenv "heartless/internal/domain/environment"
	domainws "heartless/internal/domain/workspace"
)

// newTestServices opens the app's services against a temporary database seeded
// with one workspace and one environment.
func newTestServices(t *testing.T) *app.Services {
	t.Helper()
	ctx := context.Background()
	app.SetDefaultDBPath(filepath.Join(t.TempDir(), "heartless.db"))
	services, err := app.Open(ctx)
	if err != nil {
		t.Fatalf("open services: %v", err)
	}
	t.Cleanup(func() { services.DB.Close() })

	ws := domainws.NewWorkspace("Cren")
	ws.ID = "ws-1"
	if err := services.Workspaces.Save(ctx, ws); err != nil {
		t.Fatalf("save workspace: %v", err)
	}
	env := domainenv.NewEnvironment("Global Variables")
	env.ID = "env-1"
	if err := services.Environments.Save(ctx, env, ws.ID); err != nil {
		t.Fatalf("save environment: %v", err)
	}
	return services
}

// settle advances the clock past the router's slide so the leaving page is
// gone and clicks hit the page shown.
func settle(tt *ui.Tester) {
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
}

// TestEnvironmentPageOpens checks that the environments index lists the
// workspace's environments and opens one.
func TestEnvironmentPageOpens(t *testing.T) {
	services := newTestServices(t)
	tt := ui.NewTester(func(c *ui.Context) { App(c, services) }, 1000, 700)

	if !tt.HasText("Heartless") || !tt.HasText("Cren") {
		t.Fatalf("title bar texts %q", tt.Texts())
	}

	if err := tt.Click("No environment"); err != nil {
		t.Fatalf("open environments: %v", err)
	}
	settle(tt)
	if !tt.HasText("Manage the variables each environment sends with your requests.") || !tt.HasText("Global Variables") {
		t.Fatalf("index not shown, texts %q", tt.Texts())
	}

	if err := tt.Click("Global Variables"); err != nil {
		t.Fatalf("open environment: %v", err)
	}
	settle(tt)
	if !tt.HasText("Variables defined here are available while this environment is active.") {
		t.Fatalf("detail not shown, texts %q", tt.Texts())
	}
}

// TestEnvironmentVariableCreate adds a variable on an environment's page and
// confirms it is stored scoped to that environment.
func TestEnvironmentVariableCreate(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	s := &shellState{workspaceID: "ws-1", environmentID: "env-1", newVarName: "baseUrl", newVarValue: "https://x.test"}
	s.createEnvironmentVariable(services, "env-1")

	vars, err := services.Variables.List(ctx, "", "env-1")
	if err != nil {
		t.Fatalf("list variables: %v", err)
	}
	if len(vars) != 1 {
		t.Fatalf("variables = %d, want 1", len(vars))
	}
	if vars[0].Name != "baseUrl" || vars[0].Value != "https://x.test" || vars[0].EnvironmentID != "env-1" {
		t.Fatalf("variable = %+v", *vars[0])
	}
}

// TestEnvironmentCardCount checks the index shows the variable count per
// environment.
func TestEnvironmentCardCount(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	s := &shellState{workspaceID: "ws-1", newVarName: "baseUrl", newVarValue: "https://x.test"}
	s.createEnvironmentVariable(services, "env-1")

	st := shellState{workspaceID: "ws-1", router: ui.NewRouter("/environments")}
	st.load(ctx, services)
	if got := len(st.variablesForEnvironment("env-1")); got != 1 {
		t.Fatalf("variables for env = %d, want 1", got)
	}
	if got := plural(1, "variable", "variables"); got != "1 variable" {
		t.Fatalf("plural = %q", got)
	}
}

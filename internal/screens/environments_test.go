package screens

import (
	"context"
	"path/filepath"
	"testing"

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

// TestEnvironmentPageOpens checks the title bar breadcrumb and that the
// environments list opens an environment's detail page.
func TestEnvironmentPageOpens(t *testing.T) {
	services := newTestServices(t)
	tt := ui.NewTester(func(c *ui.Context) { App(c, services) }, 1000, 700)

	if !tt.HasText("Heartless") || !tt.HasText("Cren") {
		t.Fatalf("title bar texts %q", tt.Texts())
	}

	// Open the environments index from the environment label.
	if err := tt.Click("No environment"); err != nil {
		// A first workspace is auto-selected; the label reads the active env.
		if err := tt.Click("Global Variables"); err != nil {
			t.Fatalf("open environments: %v", err)
		}
	}
	if !tt.HasText("Manage the variables each environment sends with your requests.") {
		t.Fatalf("index not shown, texts %q", tt.Texts())
	}

	// Open the environment card.
	if err := tt.Click("Open"); err != nil {
		t.Fatalf("open environment: %v", err)
	}
	if !tt.HasText("Variables defined here are available while this environment is active.") {
		t.Fatalf("detail not shown, texts %q", tt.Texts())
	}
}

// TestEnvironmentVariableCreate adds a variable on an environment's page and
// confirms it is stored scoped to that environment.
func TestEnvironmentVariableCreate(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	st := shellState{router: ui.NewRouter("/environments/env-1")}
	st.load(ctx, services)
	tt := ui.NewTester(func(c *ui.Context) { App(c, services) }, 1000, 700)
	_ = tt

	// Drive the state helper directly: it is what the Add button calls.
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

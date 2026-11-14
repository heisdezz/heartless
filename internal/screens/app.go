package screens

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domainenv "heartless/internal/domain/environment"
	domainvar "heartless/internal/domain/variable"
	domainws "heartless/internal/domain/workspace"
)

// varDraft holds the editable name and value of one variable. Drafts live in
// shellState so typed edits survive across builds until they are saved.
type varDraft struct {
	Name  string
	Value string
}

// shellState holds mutable state for the shell UI.
type shellState struct {
	sidebarWidth    float32
	sidebarExpanded bool
	workspaceID     string
	environmentID   string
	router          *ui.Router

	workspaces   []domainws.Workspace
	environments []domainenv.Environment
	variables    []domainvar.Variable

	loaded bool
	err    string

	showWorkspaceDialog bool
	showCreateWorkspace bool
	newWorkspaceName    string

	showCreateEnv bool
	newEnvName    string

	newVarName  string
	newVarValue string
	varDrafts   map[string]*varDraft

	nextID int
}

func (s *shellState) localID(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s-%d", prefix, s.nextID)
}

func (s *shellState) currentWorkspace() (domainws.Workspace, bool) {
	for _, ws := range s.workspaces {
		if ws.ID == s.workspaceID {
			return ws, true
		}
	}
	return domainws.Workspace{}, false
}

func (s *shellState) workspaceEnvironments() []domainenv.Environment {
	out := make([]domainenv.Environment, len(s.environments))
	copy(out, s.environments)
	return out
}

func (s *shellState) environmentByID(id string) (*domainenv.Environment, bool) {
	for i := range s.environments {
		if s.environments[i].ID == id {
			return &s.environments[i], true
		}
	}
	return nil, false
}

func (s *shellState) currentEnvironment() (domainenv.Environment, bool) {
	for _, env := range s.environments {
		if env.ID == s.environmentID {
			return env, true
		}
	}
	return domainenv.Environment{}, false
}

// variablesForEnvironment returns the variables scoped to envID.
func (s *shellState) variablesForEnvironment(envID string) []domainvar.Variable {
	var out []domainvar.Variable
	for _, v := range s.variables {
		if v.EnvironmentID == envID {
			out = append(out, v)
		}
	}
	return out
}

// varDraft returns the draft for v, seeding it from the stored variable the
// first time it is seen.
func (s *shellState) varDraft(v domainvar.Variable) *varDraft {
	if s.varDrafts == nil {
		s.varDrafts = make(map[string]*varDraft)
	}
	d, ok := s.varDrafts[v.ID]
	if !ok {
		d = &varDraft{Name: v.Name, Value: v.Value}
		s.varDrafts[v.ID] = d
	}
	return d
}

// openEnvironment selects envID and opens its detail page.
func (s *shellState) openEnvironment(envID string) {
	s.environmentID = envID
	s.router.Push("/environments/" + envID)
}

// selectWorkspace makes id the active workspace, resetting the environment.
func (s *shellState) selectWorkspace(id string, services *app.Services) {
	s.workspaceID = id
	s.environmentID = ""
	if err := s.reloadEnvironments(context.Background(), services); err != nil {
		s.err = err.Error()
	}
	s.router.Replace("/")
}

func (s *shellState) load(ctx context.Context, services *app.Services) {
	if s.loaded || services == nil {
		return
	}
	s.loaded = true
	s.err = ""
	if err := s.reload(ctx, services); err != nil {
		s.err = err.Error()
	}
}

func (s *shellState) reload(ctx context.Context, services *app.Services) error {
	if services == nil {
		return nil
	}
	wss, err := services.Workspaces.List(ctx, "")
	if err != nil {
		return err
	}
	s.workspaces = s.workspaces[:0]
	for _, ws := range wss {
		if ws != nil {
			s.workspaces = append(s.workspaces, *ws)
		}
	}
	if len(s.workspaces) > 0 {
		found := false
		for _, ws := range s.workspaces {
			if ws.ID == s.workspaceID {
				found = true
				break
			}
		}
		if !found {
			s.workspaceID = s.workspaces[0].ID
		}
	} else {
		s.workspaceID = ""
	}
	return s.reloadEnvironments(ctx, services)
}

func (s *shellState) reloadEnvironments(ctx context.Context, services *app.Services) error {
	if services == nil || s.workspaceID == "" {
		s.environments = nil
		s.environmentID = ""
		s.variables = nil
		return nil
	}
	envs, err := services.Environments.List(ctx, s.workspaceID)
	if err != nil {
		return err
	}
	s.environments = s.environments[:0]
	for _, env := range envs {
		if env != nil {
			s.environments = append(s.environments, *env)
		}
	}
	found := false
	for _, env := range s.environments {
		if env.ID == s.environmentID {
			found = true
			break
		}
	}
	if !found {
		s.environmentID = ""
	}
	return s.reloadVariables(ctx, services)
}

func (s *shellState) reloadVariables(ctx context.Context, services *app.Services) error {
	if services == nil || s.workspaceID == "" {
		s.variables = nil
		return nil
	}
	vars, err := services.Variables.List(ctx, s.workspaceID, "")
	if err != nil {
		return err
	}
	s.variables = s.variables[:0]
	for _, v := range vars {
		if v != nil {
			s.variables = append(s.variables, *v)
		}
	}
	return nil
}

// App renders the main application view.
func App(c *ui.Context, services *app.Services) {
	tk := theme.TokensOf(c)
	root := ui.Column(c).Fill().Background(tk.BgLayout)

	st := ui.Local(root, "shell", func() shellState {
		return shellState{
			sidebarWidth:    260,
			sidebarExpanded: true,
			router:          ui.NewRouter("/"),
		}
	})
	st.load(context.Background(), services)

	root.Children(func() {
		titleBar(c, tk, st, services)
		ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
			sidebar(c, st, tk, services)
			mainContent(c, tk, st, services)
		})

		workspaceDialog(c, st, services)
		createWorkspaceModal(c, st, services)
		createEnvironmentModal(c, st, services)
	})
}

func mainContent(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Background(tk.BgContainer).Children(func() {
		st.router.View(c, func(r *ui.Route) {
			switch {
			case r.Match("/"):
				r.Title("Home")
				workspaceView(c, tk, st)
			case r.Match("/settings"):
				r.Title("Settings")
				settingsView(c, tk)
			case r.Match("/environments"):
				r.Title("Environments")
				environmentsView(c, tk, st, services)
			case r.Match("/environments/{id}"):
				r.Title("Environment")
				environmentDetailView(c, tk, st, services, r.Param("id"))
			case r.Match("/variables"):
				r.Title("Variables")
				variablesView(c, tk, st, services)
			default:
				r.Title("Not found")
				notFoundView(c, tk, r.Path())
			}
		})
	})
}

func workspaceView(c *ui.Context, tk *theme.Tokens, st *shellState) {
	ui.Column(c).Fill().Children(func() {
		title := "No request selected"
		if ws, ok := st.currentWorkspace(); ok {
			title = ws.Name
			if env, ok := st.currentEnvironment(); ok {
				title += " • " + env.Name
			}
		}
		contextHeader(c, tk, title)

		ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
			hint := "Select or create a request to start."
			if st.workspaceID == "" {
				hint = "Create or select a workspace to start."
			}
			pane(c, tk, "Request", hint)
			ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
			pane(c, tk, "Response", "Send a request to see the response.")
		})
	})
}

// variablesView lists workspace-scoped variables and creates new ones.
func variablesView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	ui.Column(c).Fill().Children(func() {
		pageHeader(c, tk, "Global variables", "Available to every environment in this workspace.")
		ui.Scroll(c).Grow(1).Basis(0).Children(func() {
			ui.Column(c).Padding(24, 24).Gap(8).Children(func() {
				if st.workspaceID == "" {
					emptyState(c, tk, lucide.Pencil(), "No workspace selected", "Select a workspace first.")
					return
				}
				if len(st.variables) == 0 {
					ui.Text(c, "No workspace variables yet.").TextColor(tk.TextSecondary)
				}
				for _, v := range st.variables {
					if v.EnvironmentID != "" {
						continue
					}
					label := v.Name
					if v.Value != "" {
						label += " = " + v.Value
					}
					ui.Text(c, label)
				}
				ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(12, 0).Children(func() {
					field.Input(c, &st.newVarName, field.InputProps{Placeholder: "Name"}).Grow(3).Basis(0)
					field.Input(c, &st.newVarValue, field.InputProps{Placeholder: "Value"}).Grow(5).Basis(0)
					if action.Button(c, "Create", action.ButtonProps{Variant: action.Primary}).Clicked() {
						st.createVariable(services)
					}
				})
			})
		})
	})
}

func settingsView(c *ui.Context, tk *theme.Tokens) {
	ui.Column(c).Fill().Children(func() {
		pageHeader(c, tk, "Settings", "")
		ui.Column(c).Grow(1).Basis(0).Padding(24).Gap(8).Children(func() {
			ui.Text(c, "Application settings").FontSize(16).Bold()
			ui.Text(c, "Settings UI not implemented yet.").TextColor(tk.TextSecondary)
		})
	})
}

func notFoundView(c *ui.Context, tk *theme.Tokens, path string) {
	ui.Column(c).Fill().Children(func() {
		contextHeader(c, tk, "Not found")
		ui.Column(c).Grow(1).Basis(0).Center().Gap(8).Padding(24).Children(func() {
			ui.Text(c, "Page not found").Bold()
			ui.Text(c, path).TextColor(tk.TextSecondary)
		})
	})
}

func contextHeader(c *ui.Context, tk *theme.Tokens, title string) {
	ui.Row(c).
		Height(40).
		Padding(0, 16).
		AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.BorderSecondary).
		Background(tk.BgContainer).
		Children(func() {
			ui.Text(c, title).TextColor(tk.TextSecondary)
		})
}

func pane(c *ui.Context, tk *theme.Tokens, title, hint string) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
		ui.Row(c).
			Height(36).
			Padding(0, 12).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(tk.BorderSecondary).
			Children(func() {
				ui.Text(c, title).FontSize(13).Bold()
			})

		ui.Column(c).Grow(1).Basis(0).Center().Gap(6).Padding(24).Children(func() {
			ui.Text(c, hint).TextColor(tk.TextSecondary)
		})
	})
}

func (s *shellState) createEnvironment(services *app.Services) {
	name := strings.TrimSpace(s.newEnvName)
	if s.workspaceID == "" || name == "" || services == nil {
		return
	}
	env := domainenv.NewEnvironment(name)
	env.ID = s.localID("env")
	if err := services.Environments.Save(context.Background(), env, s.workspaceID); err != nil {
		s.err = err.Error()
		return
	}
	s.environmentID = env.ID
	s.newEnvName = ""
	s.err = ""
	if err := s.reloadEnvironments(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// createEnvironmentVariable adds a variable scoped to envID from the new
// variable inputs.
func (s *shellState) createEnvironmentVariable(services *app.Services, envID string) {
	name := strings.TrimSpace(s.newVarName)
	if s.workspaceID == "" || name == "" || services == nil {
		return
	}
	v := domainvar.NewVariable(name, s.newVarValue, "text", false)
	v.ID = s.localID("var")
	v.Scope = "environment"
	v.WorkspaceID = s.workspaceID
	v.EnvironmentID = envID
	if err := services.Variables.Save(context.Background(), v.ID, s.workspaceID, envID, v); err != nil {
		s.err = err.Error()
		return
	}
	s.newVarName = ""
	s.newVarValue = ""
	s.err = ""
	if err := s.reloadVariables(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// saveVariable persists an edited environment variable.
func (s *shellState) saveVariable(services *app.Services, envID, varID, name, value string) {
	if services == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	v := domainvar.NewVariable(name, value, "text", false)
	v.ID = varID
	v.Scope = "environment"
	v.WorkspaceID = s.workspaceID
	v.EnvironmentID = envID
	if err := services.Variables.Save(context.Background(), varID, s.workspaceID, envID, v); err != nil {
		s.err = err.Error()
		return
	}
	s.err = ""
	if err := s.reloadVariables(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

func (s *shellState) createVariable(services *app.Services) {
	name := strings.TrimSpace(s.newVarName)
	if s.workspaceID == "" || name == "" || services == nil {
		return
	}
	v := domainvar.NewVariable(name, s.newVarValue, "text", false)
	v.ID = s.localID("var")
	v.Scope = "workspace"
	v.WorkspaceID = s.workspaceID
	if err := services.Variables.Save(context.Background(), v.ID, s.workspaceID, "", v); err != nil {
		s.err = err.Error()
		return
	}
	s.newVarName = ""
	s.newVarValue = ""
	s.err = ""
	if err := s.reloadVariables(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

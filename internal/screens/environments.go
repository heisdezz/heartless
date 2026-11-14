package screens

import (
	"context"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domainenv "heartless/internal/domain/environment"
	domainvar "heartless/internal/domain/variable"
)

// environmentsView is the environments index: a page header plus a card list of
// the current workspace's environments.
func environmentsView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	ui.Column(c).Fill().Children(func() {
		pageHeader(c, tk, "Environments", "Manage the variables each environment sends with your requests.")
		ui.Scroll(c).Grow(1).Basis(0).Children(func() {
			ui.Column(c).WidthPercent(100).Padding(24, 24).Gap(10).Children(func() {
				if _, ok := st.currentWorkspace(); !ok {
					emptyState(c, tk, lucide.Database(), "No workspace selected", "Select or create a workspace to manage environments.")
					return
				}

				envs := st.workspaceEnvironments()
				if len(envs) == 0 {
					emptyState(c, tk, lucide.Database(), "No environments yet", "Create an environment to start defining variables.")
				}
				for i := range envs {
					environmentCard(c, tk, st, &envs[i])
				}

				ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(4, 0).Children(func() {
					field.Input(c, &st.newEnvName, field.InputProps{Placeholder: "New environment name"}).Grow(1).Basis(0)
					if action.Button(c, "Create", action.ButtonProps{Variant: action.Primary}).Clicked() {
						st.createEnvironment(services)
					}
				})
			})
		})
	})
}

// environmentCard is one row of the environments list; clicking it opens the
// environment's variable editor.
func environmentCard(c *ui.Context, tk *theme.Tokens, st *shellState, env *domainenv.Environment) {
	active := env.ID == st.environmentID
	row := ui.Row(c)
	row.Gap(12).Height(56).MinWidth(0).Shrink(1).Radius(tk.BorderRadius).
		Border(1, tk.BorderSecondary).PaddingX(14).AlignItems(ui.Center).
		Background(tk.BgContainer).Cursor(ui.CursorPointer)
	if row.Hovered() {
		row.Background(tk.FillSecondary)
	}
	row.Children(func() {
		ui.Box(c).Width(28).Height(28).Center().Radius(tk.BorderRadiusSM).Background(tk.FillSecondary).Children(func() {
			ui.Icon(c, lucide.Database()).FontSize(14).TextColor(tk.TextSecondary)
		})

		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			label := env.Name
			if active {
				label += "   Active"
			}
			ui.Text(c, label).FontWeight(500).SingleLine()
			n := len(st.variablesForEnvironment(env.ID))
			ui.Text(c, plural(n, "variable", "variables")).FontSize(12).TextColor(tk.TextTertiary).SingleLine()
		})

		if action.Button(c, "Open", action.ButtonProps{Variant: action.Text}).Clicked() {
			st.openEnvironment(env.ID)
		}
	})
	if row.Clicked() {
		st.openEnvironment(env.ID)
	}
}

// environmentDetailView is the inner page for one environment, where its
// variables are listed and set.
func environmentDetailView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, envID string) {
	env, ok := st.environmentByID(envID)
	if !ok {
		ui.Column(c).Fill().Children(func() {
			pageHeader(c, tk, "Environment", "")
			emptyState(c, tk, lucide.Database(), "Environment not found", "It may have been deleted.")
		})
		return
	}

	ui.Column(c).Fill().Children(func() {
		pageHeader(c, tk, env.Name, "Variables defined here are available while this environment is active.")
		ui.Row(c).Padding(0, 24).Height(44).Gap(4).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			if action.Button(c, "All environments", action.ButtonProps{Variant: action.Text}).Clicked() {
				st.router.Replace("/environments")
			}
			ui.Box(c).Grow(1)
			if env.ID != st.environmentID {
				if action.Button(c, "Set as active", action.ButtonProps{}).Clicked() {
					st.environmentID = env.ID
				}
			}
		})

		ui.Scroll(c).Grow(1).Basis(0).Children(func() {
			ui.Column(c).Padding(24, 24).Gap(6).Children(func() {
				vars := st.variablesForEnvironment(env.ID)
				ui.Row(c).Height(28).PaddingX(2).Children(func() {
					ui.Text(c, "Name").Grow(3).Basis(0).FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
					ui.Text(c, "Value").Grow(5).Basis(0).FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
					ui.Box(c).Width(64)
				})
				if len(vars) == 0 {
					ui.Text(c, "No variables yet — add one below.").TextColor(tk.TextSecondary).Padding(6, 0)
				}
				for _, v := range vars {
					v := v
					variableRow(c, tk, st, services, env.ID, v)
				}

				ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(12, 0).Children(func() {
					field.Input(c, &st.newVarName, field.InputProps{Placeholder: "Name"}).Grow(3).Basis(0)
					field.Input(c, &st.newVarValue, field.InputProps{Placeholder: "Value"}).Grow(6).Basis(0)
					if action.Button(c, "Add", action.ButtonProps{Variant: action.Primary}).Clicked() {
						st.createEnvironmentVariable(services, env.ID)
					}
				})
			})
		})
	})
}

// variableRow renders one editable variable in an environment. The draft's
// name and value live in shellState so that typing is retained across builds.
func variableRow(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, envID string, v domainvar.Variable) {
	d := st.varDraft(v)
	name := field.Input(c, &d.Name, field.InputProps{}).Grow(3).Basis(0)
	if name.Changed() {
		st.saveVariable(services, envID, v.ID, d.Name, d.Value)
	}
	value := field.Input(c, &d.Value, field.InputProps{Variant: field.InputFilled}).Grow(6).Basis(0)
	if value.Changed() {
		st.saveVariable(services, envID, v.ID, d.Name, d.Value)
	}
	if action.Button(c, "Delete", action.ButtonProps{Variant: action.Text}).Clicked() {
		if err := services.Variables.Delete(context.Background(), v.ID); err != nil {
			st.err = err.Error()
		} else {
			delete(st.varDrafts, v.ID)
			_ = st.reloadVariables(context.Background(), services)
		}
	}
}

// emptyState centers an icon, title and hint.
func emptyState(c *ui.Context, tk *theme.Tokens, icon *ui.SVG, title, hint string) {
	ui.Column(c).Padding(48).Gap(6).Center().Children(func() {
		ui.Icon(c, icon).FontSize(28).TextColor(tk.TextTertiary)
		ui.Text(c, title).FontSize(15).FontWeight(600)
		ui.Text(c, hint).FontSize(13).TextColor(tk.TextSecondary)
	})
}

// pageHeader renders a page title with a subtitle.
func pageHeader(c *ui.Context, tk *theme.Tokens, title, subtitle string) {
	ui.Column(c).Padding(24, 16).Gap(2).BorderWidth(0, 0, 1, 0).
		BorderColor(tk.BorderSecondary).Children(func() {
		ui.Text(c, title).FontSize(18).FontWeight(600)
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(13).TextColor(tk.TextSecondary)
		}
	})
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

package screens

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domainws "heartless/internal/domain/workspace"
)

// workspaceDialog shows the workspace and environment chooser.
func workspaceDialog(c *ui.Context, st *shellState, services *app.Services) {
	if !feedback.Modal(c, &st.showWorkspaceDialog, feedback.ModalProps{
		Title:    "Workspaces",
		Width:    520,
		NoFooter: true,
	}, func() {
		tk := theme.TokensOf(c)
		ui.Column(c).Gap(12).Children(func() {
			if len(st.workspaces) == 0 {
				ui.Text(c, "No workspaces yet.").TextColor(tk.TextSecondary)
			}
			ui.Column(c).Gap(2).Children(func() {
				for _, ws := range st.workspaces {
					ws := ws
					label := ws.Name
					active := ws.ID == st.workspaceID
					if active {
						label += " (active)"
					}
					if action.Button(c, label, action.ButtonProps{Variant: action.Text}).Clicked() {
						st.selectWorkspace(ws.ID, services)
						st.showWorkspaceDialog = false
					}
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				if action.Button(c, "New workspace", action.ButtonProps{Variant: action.Primary}).Clicked() {
					st.newWorkspaceName = ""
					st.showCreateWorkspace = true
				}
				if st.workspaceID != "" {
					if action.Button(c, "New environment", action.ButtonProps{}).Clicked() {
						st.newEnvName = ""
						st.showCreateEnv = true
					}
				}
			})
			if st.workspaceID != "" {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "Environments").Bold()
					envs := st.workspaceEnvironments()
					if len(envs) == 0 {
						ui.Text(c, "No environments.").TextColor(tk.TextSecondary)
					}
					for _, env := range envs {
						env := env
						label := env.Name
						if env.ID == st.environmentID {
							label += " (active)"
						}
						if action.Button(c, label, action.ButtonProps{Variant: action.Text}).Clicked() {
							st.setActiveEnvironment(context.Background(), services, env.ID)
							st.showWorkspaceDialog = false
						}
					}
				})
			}
		})
	}) {
		// Footer hidden; OK is unused.
	}
}

// createWorkspaceModal creates a workspace from the typed name.
func createWorkspaceModal(c *ui.Context, st *shellState, services *app.Services) {
	if feedback.Modal(c, &st.showCreateWorkspace, feedback.ModalProps{
		Title:  "Create workspace",
		Width:  420,
		OkText: "Create",
	}, func() {
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, "Name")
			field.Input(c, &st.newWorkspaceName, field.InputProps{Placeholder: "My workspace"})
		})
	}) {
		name := strings.TrimSpace(st.newWorkspaceName)
		if name != "" && services != nil {
			// Keep the generated unique ID: a local counter would repeat across
			// launches and overwrite existing rows on save.
			ws := domainws.NewWorkspace(name)
			if err := services.Workspaces.Save(context.Background(), ws); err != nil {
				st.err = err.Error()
			} else {
				st.workspaceID = ws.ID
				st.environmentID = ""
				st.err = ""
				if err := st.reload(context.Background(), services); err != nil {
					st.err = err.Error()
				} else {
					_ = st.reloadEnvironments(context.Background(), services)
				}
			}
		}
		st.newWorkspaceName = ""
		st.showCreateWorkspace = false
	}
}

// createEnvironmentModal lists an environment's peers and lets the user add one.
func createEnvironmentModal(c *ui.Context, st *shellState, services *app.Services) {
	if st.workspaceID == "" {
		st.showCreateEnv = false
		return
	}
	if feedback.Modal(c, &st.showCreateEnv, feedback.ModalProps{
		Title:      "New environment",
		Width:      420,
		OkText:     "Create",
		CancelText: "Close",
	}, func() {
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, "Name").FontSize(13).TextColor(theme.TokensOf(c).TextSecondary)
			field.Input(c, &st.newEnvName, field.InputProps{Placeholder: "Development"})
		})
	}) {
		st.createEnvironment(services)
		st.showCreateEnv = false
	}
}

// importCollectionModal imports a collection from a URL or picked/dropped file.
func importCollectionModal(c *ui.Context, st *shellState, services *app.Services) {
	if feedback.Modal(c, &st.showImportCollection, feedback.ModalProps{
		Title:      "Import collection",
		Width:      560,
		NoFooter:   true,
		CancelText: "Close",
	}, func() {
		tk := theme.TokensOf(c)
		ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			if st.importCollectionBusy {
				ui.Text(c, "Importing…").TextColor(tk.TextSecondary)
			}
			ui.Text(c, "Import from URL").FontSize(13).TextColor(tk.TextSecondary)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				field.Input(c, &st.importCollectionURL, field.InputProps{Placeholder: "https://…/collection.json"}).Grow(1).Basis(0).MinWidth(0)
				disabled := st.importCollectionBusy || strings.TrimSpace(st.importCollectionURL) == "" || st.workspaceID == ""
				btn := action.Button(c, "Import URL", action.ButtonProps{Variant: action.Primary}).Shrink(0)
				if disabled {
					btn.Disabled(true).Opacity(0.5)
				}
				if btn.Clicked() && !disabled {
					st.importCollectionFromURL(services, strings.TrimSpace(st.importCollectionURL))
				}
			})

			ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)

			ui.Text(c, "Import from file").FontSize(13).TextColor(tk.TextSecondary)
			field.Upload(c, field.UploadProps{
				Accept:      ".json,application/json,text/json",
				Multiple:    false,
				Dragger:     true,
				Title:       "Click or drag a collection file here",
				Description: "Supports Postman and Insomnia JSON exports",
				Disabled:    st.importCollectionBusy || st.workspaceID == "",
				OnFiles: func(files []field.File) {
					if len(files) == 0 {
						return
					}
					st.importCollectionFromFile(services, files[0])
				},
			}, nil)

			if st.importCollectionErr != "" {
				ui.Text(c, st.importCollectionErr).FontSize(12).TextColor(tk.Error)
			}
			if st.workspaceID == "" {
				ui.Text(c, "Select a workspace first.").FontSize(12).TextColor(tk.TextTertiary)
			}
		})
	}) {
		// NoFooter: no default actions.
	}
}

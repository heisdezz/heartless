package screens

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domainreq "heartless/internal/domain/request"
)

// Workspace settings sections, in the order the sidebar lists them.
const (
	wsSectionAuth = iota
	wsSectionVariables
	wsSectionEnvironments
)

var workspaceSettingsSections = []string{"Auth", "Variables", "Environments"}

// workspaceSettingsView is the workspace's own settings page: the workspace's
// default auth with auto-inherit, its variables, and its environments. The
// title bar's Workspace Settings entry and Cmd+; land here.
func workspaceSettingsView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	st.ensureWorkspaceSettings()

	ui.Column(c).Fill().MinWidth(0).Children(func() {
		subtitle := "Authentication, variables and environments for this workspace."
		if ws, ok := st.currentWorkspace(); ok {
			subtitle = "Settings that apply to every request in " + ws.Name + "."
		}
		pageHeader(c, tk, "Workspace Settings", subtitle)

		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
			// Section list, styled as the app settings page's is.
			ui.Column(c).Width(220).Shrink(0).Padding(12, 12).Gap(2).
				AlignItems(ui.Stretch).BorderWidth(0, 1, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
				for i, name := range workspaceSettingsSections {
					section := i
					btn := action.Button(c, name, action.ButtonProps{Variant: action.Text}).
						Justify(ui.Start).AlignSelf(ui.Stretch)
					if st.wsSettingsSection == section {
						btn.Background(tk.FillSecondary).FontWeight(600).TextColor(tk.Text)
					} else {
						btn.TextColor(tk.TextSecondary)
					}
					btn.OnClick(func() { st.wsSettingsSection = section })
				}
			})

			ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				ui.Column(c).AlignSelf(ui.Stretch).Grow(1).MinWidth(0).Padding(24).Children(func() {
					ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(720).Gap(16).Children(func() {
						switch st.wsSettingsSection {
						case wsSectionAuth:
							workspaceAuthSettings(c, tk, st, services)
						case wsSectionVariables:
							variablesBody(c, tk, st, services)
						case wsSectionEnvironments:
							environmentsBody(c, tk, st, services)
						}
					})
				})
			})
		})
	})
}

// workspaceAuthSettings edits the workspace's default auth: an auto-inherit
// switch and the credentials themselves, drawn with the request editor's auth
// panel so the two look and behave alike.
func workspaceAuthSettings(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	settingsGroup(c, tk, "Authentication", "Credentials this workspace's requests can share.")(func() {
		rowSetting(c, tk, "Auto-inherit", "Requests without their own auth send these credentials.")(func() {
			choice.Switch(c, &st.wsAutoInherit, choice.SwitchProps{})
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Scheme", "The auth type requests inherit from this workspace.")(func() {
			workspaceAuthTypeSelect(c, tk, st)
		})
	})

	// The credential fields for the chosen scheme, as the request editor draws
	// them.
	authFields(c, tk, &st.wsAuth)
	if st.wsAutoInherit && st.wsAuth.Type == domainreq.AuthNone {
		authHint(c, tk, "Auto-inherit is on, but No Auth sends no credentials.")
	}

	// The credentials save through the Save bar, the way app settings do.
	ui.Row(c).Height(space(c, 44)).AlignItems(ui.Center).Gap(8).Children(func() {
		save := action.Button(c, "Save", action.ButtonProps{Variant: action.Primary, Size: theme.Small}).Shrink(0)
		if !st.workspaceSettingsDirty() {
			save.Disabled(true)
		}
		if save.Clicked() && st.workspaceSettingsDirty() {
			st.saveWorkspaceSettings(services)
		}
		switch {
		case st.workspaceSettingsDirty():
			ui.Text(c, "Unsaved changes").FontSize(textSize(c, 12)).TextColor(tk.TextTertiary).SingleLine()
		case st.wsSettingsSaved:
			ui.Text(c, "Saved").FontSize(textSize(c, 12)).TextColor(tk.Success).SingleLine()
		}
		ui.Box(c).Grow(1)
	})
}

// workspaceAuthTypeSelect is the scheme menu for the workspace's default auth.
// There is no parent above a workspace, so Inherit is left out.
func workspaceAuthTypeSelect(c *ui.Context, tk *theme.Tokens, st *shellState) {
	if st.wsAuth.Type == "" {
		st.wsAuth.Type = domainreq.AuthNone
	}
	bar := ui.Row(c)
	open := ui.Local(bar, "wsAuthMenu", func() bool { return false })
	bar.AlignItems(ui.Center).Children(func() {
		ui.Box(c).Shrink(0).Children(func() {
			chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{
				Items: authMenuItemsWith(st.wsAuth.Type, false),
			}, func(t ui.Element) {
				authTrigger(t, c, tk, &st.wsAuth)
			})
			if chosen != "" {
				st.wsAuth.Type = domainreq.AuthType(chosen)
			}
		})
	})
}

// workspaceSettingsDirty reports whether the auth draft differs from the
// workspace's stored one. An unset stored type is the same as No Auth, the
// value the draft seeds with.
func (s *shellState) workspaceSettingsDirty() bool {
	ws, ok := s.currentWorkspace()
	if !ok {
		return false
	}
	base := ws.DefaultAuth
	if base.Type == "" {
		base.Type = domainreq.AuthNone
	}
	return s.wsAuth != base || s.wsAutoInherit != ws.AutoInheritAuth
}

// ensureWorkspaceSettings seeds the auth draft from the current workspace, and
// reseeds it when the active workspace changes.
func (s *shellState) ensureWorkspaceSettings() {
	if s.wsSettingsFor == s.workspaceID {
		return
	}
	s.wsSettingsFor = s.workspaceID
	s.wsSettingsSaved = false
	s.wsAuth = domainreq.Auth{}
	s.wsAutoInherit = false
	if ws, ok := s.currentWorkspace(); ok {
		s.wsAuth = ws.DefaultAuth
		if s.wsAuth.Type == "" {
			s.wsAuth.Type = domainreq.AuthNone
		}
		s.wsAutoInherit = ws.AutoInheritAuth
	}
}

// saveWorkspaceSettings writes the auth draft to the workspace and stores it.
func (s *shellState) saveWorkspaceSettings(services *app.Services) {
	ws, ok := s.currentWorkspace()
	if !ok || services == nil {
		return
	}
	ctx := context.Background()
	stored, err := services.Workspaces.Get(ctx, ws.ID)
	if err != nil {
		s.err = err.Error()
		return
	}
	stored.DefaultAuth = s.wsAuth
	stored.AutoInheritAuth = s.wsAutoInherit
	if err := services.Workspaces.Save(ctx, stored); err != nil {
		s.err = err.Error()
		return
	}
	for i := range s.workspaces {
		if s.workspaces[i].ID == ws.ID {
			s.workspaces[i].DefaultAuth = s.wsAuth
			s.workspaces[i].AutoInheritAuth = s.wsAutoInherit
		}
	}
	s.wsSettingsSaved = true
	s.err = ""
}

// resolveAuthInheritance fills an inheriting request from the workspace's
// default auth. An explicit Inherit always follows the workspace; an unset type
// does only when the workspace turns auto-inherit on.
func (s *shellState) resolveAuthInheritance(a domainreq.Auth) domainreq.Auth {
	switch a.Type {
	case domainreq.AuthInherit:
	case "":
		ws, ok := s.currentWorkspace()
		if !ok || !ws.AutoInheritAuth {
			return a
		}
	default:
		return a
	}
	ws, ok := s.currentWorkspace()
	if !ok || ws.DefaultAuth.Type == "" || ws.DefaultAuth.Type == domainreq.AuthInherit {
		return a
	}
	return ws.DefaultAuth
}

package screens

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
)

// titleBar renders the application title bar with a workspace/env breadcrumb.
func titleBar(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	bar := c.TitleBar()
	h := max(bar.Height, 38)

	// Workspace Settings (Cmd+;) opens the environment page.
	c.OnShortcut(ui.Cmd, ui.KeySemicolon, func() {
		if st.workspaceID != "" {
			st.router.Push("/environments")
		}
	})

	var chosen string
	row := ui.Row(c)
	wsOpen := ui.Local(row, "workspaceMenu", func() bool { return false })
	row.
		Height(h).
		Padding(0, bar.Right+12, 0, bar.Left+12).
		AlignItems(ui.Center).
		Gap(8).
		Background(tk.BgLayout).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.BorderSecondary).
		DragWindow().
		Children(func() {
			ui.Text(c, "Heartless").Bold()

			wsName := "No workspace"
			if ws, ok := st.currentWorkspace(); ok {
				wsName = ws.Name
			}
			chosen = overlay.DropdownMenu(c, wsOpen, overlay.DropdownMenuProps{Items: workspaceMenuItems(st)}, func(t ui.Element) {
				workspaceChip(t, c, tk, wsName)
			})

			ui.Icon(c, lucide.ChevronRight()).FontSize(14).TextColor(tk.TextTertiary)

			// Environment: opens the environment page to select or create one.
			envLabel := "No environment"
			if env, ok := st.currentEnvironment(); ok {
				envLabel = env.Name
			}
			if action.Button(c, envLabel, action.ButtonProps{Variant: action.Text}).Clicked() {
				st.router.Push("/environments")
			}

			ui.Box(c).Grow(1)
			if st.router.CanGoBack() {
				if action.Button(c, "Back", action.ButtonProps{Variant: action.Text}).Clicked() {
					st.router.Back()
				}
			}
		})

	switch {
	case chosen == "new-workspace":
		st.newWorkspaceName = ""
		st.showCreateWorkspace = true
	case chosen == "workspace-settings":
		st.router.Push("/environments")
	case chosen == "clear-history":
		// TODO: clear send history once history is persisted.
	default:
		if id, ok := parseWorkspaceKey(chosen); ok {
			st.selectWorkspace(id, services)
		}
	}
}

// workspaceChip styles the workspace dropdown's trigger as a pill.
func workspaceChip(t ui.Element, c *ui.Context, tk *theme.Tokens, name string) {
	t.FontSize(13).Height(24).PaddingX(8).Gap(6).Radius(tk.BorderRadiusSM).
		AlignItems(ui.Center).TextColor(tk.Text).
		Transition(ui.ElementTransition{Duration: 120 * time.Millisecond, Colors: true})
	if t.Hovered() {
		t.Background(tk.FillTertiary)
	} else {
		t.Background(tk.FillSecondary)
	}
	t.Children(func() {
		ui.Text(c, name).SingleLine()
		ui.Icon(c, lucide.ChevronDown()).FontSize(12).TextColor(tk.TextTertiary)
	})
}

// workspaceMenuItems builds the rows of the title bar's workspace dropdown.
func workspaceMenuItems(st *shellState) []overlay.MenuItem {
	items := []overlay.MenuItem{{Key: "new-workspace", Label: "New Workspace", Icon: lucide.Plus()}}
	if len(st.workspaces) > 0 {
		items = append(items, overlay.MenuItem{Divider: true})
	}
	for _, ws := range st.workspaces {
		it := overlay.MenuItem{Key: "ws:" + ws.ID, Label: ws.Name}
		if ws.ID == st.workspaceID {
			it.Icon = lucide.Check()
		}
		items = append(items, it)
	}
	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "workspace-settings", Label: "Workspace Settings", Icon: lucide.Settings(), Hotkey: "mod+;"},
		overlay.MenuItem{Key: "clear-history", Label: "Clear Send History", Icon: lucide.History(), Danger: true},
	)
	return items
}

// parseWorkspaceKey extracts the workspace ID from a "ws:<id>" menu key.
func parseWorkspaceKey(key string) (string, bool) {
	const prefix = "ws:"
	if len(key) > len(prefix) && key[:len(prefix)] == prefix {
		return key[len(prefix):], true
	}
	return "", false
}

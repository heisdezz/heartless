package screens

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
)

// TitleBarHeight is the height of the title bar the view draws in place of the
// OS one, in DIPs: the window controls sit in it. main.go passes it to the
// window, and titleBar keeps to it.
const TitleBarHeight float32 = 38

// titleBar draws the window's whole decoration in place of the OS title bar: the
// bar itself, the workspace/env breadcrumb in it, and its window buttons. The
// window is created frameless (main.go), so the view fills it.
//
// A platform that still draws its own controls over the view (macOS traffic
// lights) reports their room as c.TitleBar, and the bar keeps clear of it; where
// it does not, the bar draws its buttons at its right end, the maximize one
// marked for Windows 11's snap layouts.
func titleBar(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	bar := c.TitleBar()
	h := max(bar.Height, TitleBarHeight)
	ownControls := bar.Height == 0

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
		MinWidth(0).
		Padding(0, bar.Right, 0, bar.Left).
		AlignItems(ui.Center).
		// The bar frames the content with the window's color, as the sidebar
		// does, rather than taking the panel surface the content area has.
		Background(tk.BgLayout).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.BorderSecondary).
		DragWindow().
		Children(func() {
			// The breadcrumb, kept off the window's edges and away from the
			// controls at either end.
			ui.Row(c).Height(h).Grow(1).Basis(0).MinWidth(0).
				Padding(0, 12, 0, 12).AlignItems(ui.Center).Gap(8).
				// A press here drags the window too, as the bar does, unless the
				// press lands on a control.
				DragWindow().
				Children(func() {
					appLabel(c, tk)
					ui.Icon(c, lucide.ChevronRight()).FontSize(textSize(c, 14)).TextColor(tk.TextTertiary)

					wsName := "No workspace"
					if ws, ok := st.currentWorkspace(); ok {
						wsName = ws.Name
					}
					chosen = overlay.DropdownMenu(c, wsOpen, overlay.DropdownMenuProps{Items: workspaceMenuItems(st)}, func(t ui.Element) {
						workspaceChip(t, c, tk, wsName)
					})

					ui.Icon(c, lucide.ChevronRight()).FontSize(textSize(c, 14)).TextColor(tk.TextTertiary)

					// Environment: left-click opens the environment page; right-click
					// offers the workspace's environments to switch to, plus a link to
					// the environment settings page.
					envLabel := "No environment"
					if env, ok := st.currentEnvironment(); ok {
						envLabel = env.Name
					}
					envButton := action.Button(c, envLabel, action.ButtonProps{Variant: action.Text})
					if envButton.Clicked() {
						st.router.Push("/environments")
					}
					if chosen := overlay.ContextMenu(c, envButton, overlay.DropdownMenuProps{Items: environmentMenuItems(st)}); chosen != "" {
						switch {
						case chosen == "env-settings":
							if st.environmentID != "" {
								st.router.Push("/environments/" + st.environmentID)
							} else {
								st.router.Push("/environments")
							}
						default:
							if id, ok := parseEnvKey(chosen); ok {
								st.setActiveEnvironment(context.Background(), services, id)
							}
						}
					}

					ui.Box(c).Grow(1)
					if st.router.CanGoBack() {
						if action.Button(c, "Back", action.ButtonProps{Variant: action.Text}).Clicked() {
							st.router.Back()
						}
					}
				})

			if ownControls {
				windowControls(c, tk, st)
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
		if id, ok := parseWorkspaceKey(chosen); ok && id != st.workspaceID {
			st.selectWorkspace(id, services)
		}
	}
}

// appLabel is the app's name at the start of the title bar, dimmed with the
// window when another window has the user's input.
func appLabel(c *ui.Context, tk *theme.Tokens) {
	color := tk.Text
	if !c.WindowActive() {
		color = tk.TextTertiary
	}
	ui.Text(c, "Heartless").Bold().FontSize(textSize(c, 13)).TextColor(color).SingleLine()
}

// windowControls draws the window's buttons where the platform draws none: the
// ones Windows puts in a title bar, close last, each filling the bar's height
// and closing it off at the right.
func windowControls(c *ui.Context, tk *theme.Tokens, st *shellState) {
	ui.Row(c).Height(TitleBarHeight).AlignSelf(ui.Stretch).Children(func() {
		windowButton(c, tk, "window-minimize", "Minimize", lucide.Minus(), false).
			OnClick(func() {
				if w := currentWindow(st); w != nil {
					w.Minimize()
				}
			})
		windowButton(c, tk, "window-maximize", "Maximize", lucide.Square(), false).
			MaximizeButton().
			OnClick(func() {
				if w := currentWindow(st); w != nil {
					w.ToggleMaximize()
				}
			})
		windowButton(c, tk, "window-close", "Close", lucide.X(), true).
			OnClick(func() {
				if w := currentWindow(st); w != nil {
					w.Close()
				}
			})
	})
}

// windowButton is one window control: a glyph centered in a square filling the
// title bar's height. Close lights in the error color on hover, as Windows' do.
func windowButton(c *ui.Context, tk *theme.Tokens, key, label string, icon *ui.SVG, danger bool) ui.Element {
	btn := ui.ButtonBase(c).Key(key).Label(label).Size(TitleBarHeight, TitleBarHeight).
		Center().Cursor(ui.CursorPointer).AlignSelf(ui.Stretch).
		Transition(ui.ElementTransition{Duration: 100 * time.Millisecond, Colors: true})
	glyph := tk.TextSecondary
	switch {
	case btn.Hovered() && danger:
		btn.Background(tk.Error)
		glyph = tk.PrimaryForeground
	case btn.Hovered():
		btn.Background(tk.FillSecondary)
		glyph = tk.Text
	}
	btn.Children(func() {
		ui.Icon(c, icon).FontSize(textSize(c, 14)).TextColor(glyph)
	})
	return btn
}

// currentWindow is the window the view draws in. The focused one while the app
// has the focus; otherwise the one the view last used, the app's main window.
func currentWindow(st *shellState) *mygo.Window {
	if w := mygo.FocusedWindow(); w != nil {
		st.window = w
	}
	return st.window
}

// workspaceChip styles the workspace dropdown's trigger as a pill.
func workspaceChip(t ui.Element, c *ui.Context, tk *theme.Tokens, name string) {
	t.FontSize(textSize(c, 13)).Height(24).PaddingX(8).Gap(6).Radius(tk.BorderRadiusSM).
		AlignItems(ui.Center).TextColor(tk.Text).
		Transition(ui.ElementTransition{Duration: 120 * time.Millisecond, Colors: true})
	if t.Hovered() {
		t.Background(tk.FillTertiary)
	} else {
		t.Background(tk.FillSecondary)
	}
	t.Children(func() {
		ui.Text(c, name).SingleLine()
		ui.Icon(c, lucide.ChevronDown()).FontSize(textSize(c, 12)).TextColor(tk.TextTertiary)
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

// environmentMenuItems builds the environment switcher shown on a right-click
// of the active environment chip, with a link to environment settings.
func environmentMenuItems(st *shellState) []overlay.MenuItem {
	items := make([]overlay.MenuItem, 0, len(st.environments)+2)
	for _, env := range st.environments {
		it := overlay.MenuItem{Key: "env:" + env.ID, Label: env.Name}
		if env.ID == st.environmentID {
			it.Icon = lucide.Check()
		}
		items = append(items, it)
	}
	if len(items) > 0 {
		items = append(items, overlay.MenuItem{Divider: true})
	}
	items = append(items, overlay.MenuItem{Key: "env-settings", Label: "Environment Settings", Icon: lucide.Settings()})
	return items
}

// parseEnvKey extracts the environment ID from an "env:<id>" menu key.
func parseEnvKey(key string) (string, bool) {
	const prefix = "env:"
	if len(key) > len(prefix) && key[:len(prefix)] == prefix {
		return key[len(prefix):], true
	}
	return "", false
}

// parseWorkspaceKey extracts the workspace ID from a "ws:<id>" menu key.
func parseWorkspaceKey(key string) (string, bool) {
	const prefix = "ws:"
	if len(key) > len(prefix) && key[:len(prefix)] == prefix {
		return key[len(prefix):], true
	}
	return "", false
}

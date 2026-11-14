package screens

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
)

// sidebar renders the side navigation panel.
func sidebar(c *ui.Context, st *shellState, tk *theme.Tokens, services *app.Services) {
	layout.DraggableSideNav(c, &st.sidebarWidth, &st.sidebarExpanded, layout.DraggableSideNavProps{
		MinWidth: 72,
		MaxWidth: 420,
		Header: func(nav layout.SideNavState) {
			if !nav.Expanded {
				return
			}
			ui.Column(c).Padding(12).Gap(6).Opacity(1 - nav.Collapse).Children(func() {
				if st.err != "" {
					ui.Text(c, st.err).FontSize(12).TextColor(tk.Error)
					return
				}
				if st.workspaceID == "" {
					ui.Text(c, "Create or select a workspace.").FontSize(13).TextColor(tk.TextTertiary).Opacity(1 - nav.Collapse)
				} else {
					ui.Text(c, "No requests yet").FontSize(13).TextColor(tk.TextTertiary).Opacity(1 - nav.Collapse)
				}
			})
		},
		Footer: func(nav layout.SideNavState) {
			if !nav.Expanded {
				return
			}
			ui.Row(c).Padding(12).Gap(4).Opacity(1 - nav.Collapse).Children(func() {
				if action.Button(c, "Environments", action.ButtonProps{Variant: action.Text}).Clicked() {
					st.router.Push("/environments")
				}
				if action.Button(c, "Settings", action.ButtonProps{Variant: action.Text}).Clicked() {
					st.router.Push("/settings")
				}
			})
		},
	}, func(nav layout.SideNavState) {})
}

package screens

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/layout"
)

type dashState struct {
	width    float32
	expanded bool
}

func Dash(c *ui.Context) {
	root := ui.Row(c).Fill().AlignItems(ui.Stretch)
	st := ui.Local(root, "dash-nav", func() dashState {
		return dashState{width: 280, expanded: true}
	})

	root.Children(func() {
		layout.DraggableSideNav(c, &st.width, &st.expanded, layout.DraggableSideNavProps{
			MinWidth: 80,
		}, func(nav layout.SideNavState) {
			_ = nav
		})

		ui.Column(c).Grow(1).Basis(0)
	})
}

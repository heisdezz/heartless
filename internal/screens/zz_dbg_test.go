package screens

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domaincol "heartless/internal/domain/collection"
)

func TestDbgTree(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	col := domaincol.New("ws-1", "api")
	_ = services.Collections.SaveCollection(ctx, col)

	st := shellState{workspaceID: "ws-1", sidebarWidth: 260, sidebarExpanded: true}
	st.load(ctx, services)

	tt := ui.NewTester(func(c *ui.Context) {
		tk := theme.TokensOf(c)
		ui.Row(c).Fill().Children(func() {
			ui.Box(c).Width(260).Height(600).Children(func() {
				sidebar(c, &st, tk, services)
			})
		})
	}, 500, 700)
	tt.Frame()
	r, ok := tt.Find("api")
	t.Logf("standalone sidebar: api ok=%v X=%.0f Y=%.0f W=%.0f H=%.0f", ok, r.X, r.Y, r.W, r.H)

	tt2 := ui.NewTester(func(c *ui.Context) { App(c, services) }, 900, 700)
	tt2.Frame()
	r2, ok2 := tt2.Find("api")
	t.Logf("in App: api ok=%v X=%.0f Y=%.0f W=%.0f H=%.0f", ok2, r2.X, r2.Y, r2.W, r2.H)
}

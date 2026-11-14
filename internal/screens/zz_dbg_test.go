package screens

import (
	"context"
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
)

func TestDbgTree(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()
	col := domaincol.New("ws-1", "api")
	t.Logf("save collection: %v", services.Collections.SaveCollection(ctx, col))
	req := domainreq.New("ws-1", "Log in", domainreq.KindHTTP)
	req.CollectionID = col.ID
	t.Logf("save request: %v", services.Collections.SaveRequest(ctx, req))

	st := shellState{workspaceID: "ws-1"}
	st.load(ctx, services)
	t.Logf("state: collections=%d folders=%d requests=%d err=%q", len(st.collections), len(st.folders), len(st.requests), st.err)

	tt := ui.NewTester(func(c *ui.Context) { App(c, services) }, 900, 700)
	tt.Frame()
	for _, s := range []string{"api", "Log in", "No collections", "Search"} {
		r, ok := tt.Find(s)
		t.Logf("%q ok=%v X=%.0f Y=%.0f W=%.0f H=%.0f", s, ok, r.X, r.Y, r.W, r.H)
	}
	t.Logf("texts %q", tt.Texts())
	var _ = fmt.Sprintf
}

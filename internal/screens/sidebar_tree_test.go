package screens

import (
	"context"
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
)

// TestSidebarShowsCollectionTree checks the collection tree paints inside the
// side nav. It regressed once: the nav's content column is sized by its
// children, so a scroll that only Grows collapsed and drew nothing.
func TestSidebarShowsCollectionTree(t *testing.T) {
	services := newTestServices(t)
	col := domaincol.New("ws-1", "api")
	if err := services.Collections.SaveCollection(context.Background(), col); err != nil {
		t.Fatalf("save collection: %v", err)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		App(c, services)
	}, 900, 700)
	tt.Frame()

	if !tt.HasText("api") {
		t.Fatalf("collection missing, texts %q", tt.Texts())
	}
	// The row must be laid out (hit-testable), not merely in the tree.
	if err := tt.Click("api"); err != nil {
		t.Fatalf("collection row is not clickable at a position: %v", err)
	}

	// Rows past the scroll's first screen must keep their width too: an explicit
	// zero basis on the scroll left everything after it zero-wide.
	for i := 0; i < 12; i++ {
		r := domainreq.New("ws-1", fmt.Sprintf("request %02d", i), domainreq.KindHTTP)
		r.CollectionID = col.ID
		r.Method = "GET"
		if err := services.Collections.SaveRequest(context.Background(), r); err != nil {
			t.Fatalf("save request: %v", err)
		}
	}
	tt = ui.NewTester(func(c *ui.Context) { App(c, services) }, 900, 700)
	tt.Frame()
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("request %02d", i)
		if r, ok := tt.Find(name); !ok || r.W == 0 || r.H == 0 {
			t.Fatalf("row %q has no size: %+v", name, r)
		}
	}
}

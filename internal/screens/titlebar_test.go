package screens

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domaincol "heartless/internal/domain/collection"
)

// TestTitleBarClearsWindowControls checks the view's title bar takes the room
// the window controls need and keeps its declared height.
func TestTitleBarClearsWindowControls(t *testing.T) {
	services := newTestServices(t)
	_ = services.Collections.SaveCollection(context.Background(), domaincol.New("ws-1", "api"))

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		App(c, services)
	}, 900, 700)
	bar := ui.TitleBar{Height: TitleBarHeight, Right: 138, Bottom: TitleBarHeight}
	tt.SetTitleBar(bar)
	tt.Frame()

	title, ok := tt.Find("Heartless")
	if !ok {
		t.Fatalf("title missing, texts %q", tt.Texts())
	}
	if title.X < bar.Left+8 {
		t.Errorf("title at x=%.0f, under the left controls (%.0f)", title.X, bar.Left)
	}

	env, ok := tt.Find("No environment")
	if !ok {
		t.Fatalf("environment label missing, texts %q", tt.Texts())
	}
	if env.X+env.W > 900-bar.Right-8 {
		t.Errorf("environment ends at %.0f, under the right controls (%.0f)", env.X+env.W, 900-bar.Right)
	}
	if env.Y < 0 || env.Y+env.H > bar.Height {
		t.Errorf("environment at y=%.0f h=%.0f, outside the %.0f title bar", env.Y, env.H, bar.Height)
	}

	// The tree starts below the bar.
	tree, ok := tt.Find("api")
	if !ok {
		t.Fatalf("collection missing, texts %q", tt.Texts())
	}
	if tree.Y < bar.Bottom {
		t.Errorf("collection at y=%.0f, over the title bar (%.0f)", tree.Y, bar.Bottom)
	}
}

// TestTitleBarOwnWindowControls checks that a window whose platform draws no
// controls gets the view's own, sharing the bar's right end.
func TestTitleBarOwnWindowControls(t *testing.T) {
	services := newTestServices(t)
	_ = services.Collections.SaveCollection(context.Background(), domaincol.New("ws-1", "api"))

	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		App(c, services)
	}, 900, 700)
	tt.Frame()
	// No controls from the platform: c.TitleBar is zero.

	for _, ctl := range []struct{ label string }{
		{"Minimize"}, {"Maximize"}, {"Close"},
	} {
		r, ok := tt.Find(ctl.label)
		if !ok {
			t.Fatalf("%s missing, texts %q", ctl.label, tt.Texts())
		}
		if r.Y != 0 || r.H != TitleBarHeight {
			t.Errorf("%s at y=%.0f h=%.0f, want it filling the %.0f bar", ctl.label, r.Y, r.H, TitleBarHeight)
		}
	}
	if r, _ := tt.Find("Close"); r.X+r.W != 900 {
		t.Errorf("close ends at %.0f, want the window's right edge", r.X+r.W)
	}
	if r, _ := tt.Find("Heartless"); r.X < 8 || r.Y+r.H > TitleBarHeight {
		t.Errorf("title at %+v, want it inside the bar", r)
	}
}

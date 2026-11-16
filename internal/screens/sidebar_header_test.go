package screens

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// TestSidebarSettingsBesideSearch checks the settings control is an icon button
// next to the search box in the header, opens settings, and that the old
// footer environment link is gone.
func TestSidebarSettingsBesideSearch(t *testing.T) {
	services := newTestServices(t)
	st := shellState{
		sidebarExpanded: true,
		sidebarWidth:    260,
		router:          ui.NewRouter("/"),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		sidebar(c, &st, theme.TokensOf(c), services)
	}, 900, 700)
	tt.Frame()

	// The control is labeled for assistive technology and its tooltip, but the
	// header no longer spells "Settings" out as text.
	settings, ok := tt.Find("Settings")
	if !ok {
		t.Fatalf("settings control missing, texts %q", tt.Texts())
	}
	if settings.Y < 0 || settings.Y > 60 {
		t.Fatalf("settings not in the header: %+v", settings)
	}
	// An icon button is as wide as it is tall; a text button would be wider.
	if settings.W-settings.H > 1 {
		t.Fatalf("settings control is %vx%v, want a square icon button", settings.W, settings.H)
	}
	if tt.HasText("Environments") {
		t.Fatalf("old environment link still in sidebar, texts %q", tt.Texts())
	}

	// The search input still works and sits to the left of the settings button.
	tt.ClickAt(settings.X-60, settings.Y+settings.H/2)
	tt.Type("api")
	if st.searchQuery != "api" {
		t.Fatalf("searchQuery = %q, want api", st.searchQuery)
	}

	// Clicking the icon opens settings.
	if err := tt.Click("Settings"); err != nil {
		t.Fatalf("click settings: %v", err)
	}
	if got := st.router.Path(); got != "/settings" {
		t.Fatalf("path = %q, want /settings", got)
	}
}

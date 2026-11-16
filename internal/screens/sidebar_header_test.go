package screens

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// TestSidebarSettingsBesideSearch checks Settings sits next to the search box
// in the header and the old footer environment link is gone.
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

	settings, ok := tt.Find("Settings")
	if !ok {
		t.Fatalf("settings button missing, texts %q", tt.Texts())
	}
	if settings.Y < 0 || settings.Y > 60 {
		t.Fatalf("settings not in the header: %+v", settings)
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
}

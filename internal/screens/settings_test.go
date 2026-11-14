package screens

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/config"
)

// TestSettingsRoundTrip saves settings and loads them back from a new repo.
func TestSettingsRoundTrip(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	// Defaults before anything is saved.
	got, err := services.Settings.Load(ctx)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if got.Theme.Name != "light" || !got.Interface.ShowIcons {
		t.Fatalf("defaults = %+v", got)
	}

	s := config.DefaultApplicationSettings()
	s.Theme.Name = "dark"
	s.Theme.AccentColor = "#ff5500"
	s.Interface.CompactMode = true
	s.Interface.DoubleClickAction = "edit"
	s.Plugins.Available = []string{"git-sync"}
	s.Plugins.Enabled = []string{"git-sync"}
	if err := services.Settings.Save(ctx, s); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err = services.Settings.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Theme.Name != "dark" || got.Theme.AccentColor != "#ff5500" {
		t.Fatalf("theme = %+v", got.Theme)
	}
	if !got.Interface.CompactMode || got.Interface.DoubleClickAction != "edit" {
		t.Fatalf("interface = %+v", got.Interface)
	}
	if len(got.Plugins.Enabled) != 1 || got.Plugins.Enabled[0] != "git-sync" {
		t.Fatalf("plugins = %+v", got.Plugins)
	}

	// Saving again replaces the single row rather than adding one.
	if err := services.Settings.Save(ctx, s); err != nil {
		t.Fatalf("second save: %v", err)
	}
	var count int
	if err := services.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_settings`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("rows = %d, want 1", count)
	}
}

// TestThemeConfigAppliesAccent checks the saved accent reaches the tokens.
func TestThemeConfigAppliesAccent(t *testing.T) {
	s := config.DefaultApplicationSettings()
	s.Theme.AccentColor = "#ff5500"
	cfg := ThemeConfig(s)
	tk := cfg.Tokens(false)
	if tk.Primary.R != 255 || tk.Primary.G != 0x55 || tk.Primary.B != 0 {
		t.Fatalf("primary = %+v", tk.Primary)
	}
}

// TestParseHexColor checks the accepted formats.
func TestParseHexColor(t *testing.T) {
	if _, ok := parseHexColor("#abc"); !ok {
		t.Fatal("#abc rejected")
	}
	if _, ok := parseHexColor("0066ff"); !ok {
		t.Fatal("0066ff rejected")
	}
	if _, ok := parseHexColor("#zzz"); ok {
		t.Fatal("#zzz accepted")
	}
	if _, ok := parseHexColor(""); ok {
		t.Fatal("empty accepted")
	}
}

// TestSettingsPageRenders checks the page shows its sections and controls.
func TestSettingsPageRenders(t *testing.T) {
	services := newTestServices(t)
	st := shellState{}
	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tt.Frame()

	for _, want := range []string{"Settings", "Appearance", "Interface", "Plugins", "Save", "Accent color"} {
		if !tt.HasText(want) {
			t.Fatalf("settings missing %q, texts %q", want, tt.Texts())
		}
	}
}

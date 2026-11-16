package screens

import (
	"context"
	"strconv"
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

	if err := tt.Click("Interface"); err != nil {
		t.Fatalf("open interface section: %v", err)
	}
	for _, want := range []string{"Reduce motion", "Compact mode", "Show icons", "Font size", "Double-click action"} {
		if !tt.HasText(want) {
			t.Fatalf("interface section missing %q, texts %q", want, tt.Texts())
		}
	}
}

// TestSettingsNavIsLeftAligned checks the section list reads as a sidebar
// list: each label starts at the same left edge rather than being centered in
// its button.
func TestSettingsNavIsLeftAligned(t *testing.T) {
	services := newTestServices(t)
	st := shellState{}
	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tt.Frame()

	appearance, ok := tt.Find("Appearance")
	if !ok {
		t.Fatalf("no Appearance section, texts %q", tt.Texts())
	}
	for _, name := range []string{"Interface", "Plugins"} {
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("no %s section, texts %q", name, tt.Texts())
		}
		if r.X != appearance.X {
			t.Fatalf("%s starts at x=%v, Appearance at %v: labels are centered, not left-aligned", name, r.X, appearance.X)
		}
	}
}

// TestSettingsSaveBar saves the settings and applies the theme.
func TestSettingsSaveBar(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	// Loaded through the page, so its editor copy is the one saved.
	st := shellState{}
	st.ensureSettings(services)
	st.settings.Theme.AccentColor = "#ff5500"
	st.settingsDirty = true

	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tt.Frame()
	if !tt.HasText("Unsaved changes") {
		t.Fatalf("no unsaved marker, texts %q", tt.Texts())
	}

	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if st.settingsDirty || !st.settingsSaved {
		t.Fatalf("after save: dirty=%v saved=%v", st.settingsDirty, st.settingsSaved)
	}

	saved, err := services.Settings.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if saved.Theme.AccentColor != "#ff5500" {
		t.Fatalf("saved accent = %q", saved.Theme.AccentColor)
	}
	// App builds the theme from the saved copy; saveSettings records it.
	if tok := ThemeConfig(saved).Tokens(false); tok.Primary.R != 255 {
		t.Fatalf("saved accent not applied to the theme: %+v", tok.Primary)
	}

	// Saving again is a no-op and the marker clears.
	tt.Frame()
	if !tt.HasText("Saved") || tt.HasText("Unsaved changes") {
		t.Fatalf("marker after save, texts %q", tt.Texts())
	}
}

// TestSettingsResetToDefaults restores and stores the defaults.
func TestSettingsResetToDefaults(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	st := shellState{}
	st.ensureSettings(services)
	st.settings.Theme.Name = "dark"
	st.themeVariant = 1
	st.settings.Interface.FontSize = 18
	st.saveSettings(services)

	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tt.Frame()
	if err := tt.Click("Reset to defaults"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	saved, err := services.Settings.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if saved.Theme.Name != "light" || saved.Interface.FontSize != 0 {
		t.Fatalf("saved after reset = %+v", saved)
	}
	if st.themeVariant != 0 {
		t.Fatalf("variant = %d, want light", st.themeVariant)
	}
}

// TestInterfaceFontSizeSlider edits the font size and persists it.
func TestInterfaceFontSizeSlider(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	st := shellState{settingsSection: sectionInterface}
	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tt.Frame()
	tt.Frame()

	r, ok := tt.Find("Font size")
	if !ok {
		t.Fatalf("no font size row, texts %q", tt.Texts())
	}
	// The slider fills the row's remaining width, past the label:
	// [label][value][slider]. Click near its right end for a large size.
	labelEnd := r.X + r.W
	tt.ClickAt(labelEnd+200, r.Y+8)
	tt.Frame()
	if st.settings.Interface.FontSize <= int(minimumFontSize) || st.settings.Interface.FontSize > maximumFontSize {
		t.Fatalf("font size = %d, want %d..%d", st.settings.Interface.FontSize, minimumFontSize, maximumFontSize)
	}
	if !tt.HasText(strconv.Itoa(st.settings.Interface.FontSize)) {
		t.Fatalf("readout missing %d, texts %q", st.settings.Interface.FontSize, tt.Texts())
	}

	st.saveSettings(services)
	saved, err := services.Settings.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if saved.Interface.FontSize != st.settings.Interface.FontSize {
		t.Fatalf("saved font size = %d, want %d", saved.Interface.FontSize, st.settings.Interface.FontSize)
	}
}

// TestFontSizeReachesTokens checks the saved font size scales the theme.
func TestFontSizeReachesTokens(t *testing.T) {
	s := config.DefaultApplicationSettings()
	base := ThemeConfig(s).Tokens(false).FontSize
	s.Interface.FontSize = 18
	tk := ThemeConfig(s).Tokens(false)
	if tk.FontSize != 18 || tk.FontSize <= base {
		t.Fatalf("font size = %v, default %v", tk.FontSize, base)
	}
}

// TestFontSizeSliderMarksUnsaved checks moving the font size slider makes the
// settings dirty, so Save is enabled and the change can be persisted.
func TestFontSizeSliderMarksUnsaved(t *testing.T) {
	services := newTestServices(t)
	st := shellState{settingsSection: sectionInterface}
	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 720)
	tt.Frame()

	r, ok := tt.Find("Font size")
	if !ok {
		t.Fatalf("no font size row, texts %q", tt.Texts())
	}
	// The slider fills the row past the label; clicking partway along it
	// sets a size in range.
	tt.ClickAt(r.X+r.W+200, r.Y+8)
	if st.settings.Interface.FontSize == 0 {
		t.Fatalf("slider did not change the size")
	}
	if !st.settingsDirty {
		t.Fatalf("font size change did not mark the settings dirty")
	}
	if !tt.HasText("Unsaved changes") {
		t.Fatalf("no unsaved marker, texts %q", tt.Texts())
	}
}

// TestSavedFontSizeAppliesToApp checks a font size saved from the settings
// page reaches the running app's theme, without a restart.
func TestSavedFontSizeAppliesToApp(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	var live float32
	tt := ui.NewTester(func(c *ui.Context) {
		App(c, services)
		live = theme.TokensOf(c).FontSize
	}, 1200, 800)
	tt.Frame()
	start := live

	if err := tt.Click("Settings"); err != nil {
		t.Fatalf("open settings: %v", err)
	}
	settle(tt)
	if err := tt.Click("Interface"); err != nil {
		t.Fatalf("open interface: %v", err)
	}
	settle(tt)
	r, ok := tt.Find("Font size")
	if !ok {
		t.Fatalf("no font size row, texts %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W+200, r.Y+8)
	tt.Frame()
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save: %v", err)
	}
	settle(tt)

	saved, err := services.Settings.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if saved.Interface.FontSize <= 0 {
		t.Fatalf("font size not persisted: %+v", saved.Interface)
	}
	if live != float32(saved.Interface.FontSize) {
		t.Fatalf("live font size = %v, want %d (was %v at start)", live, saved.Interface.FontSize, start)
	}
}

// TestSavedFontSizeUsedAtStartup checks the app starts with the stored font
// size rather than the default.
func TestSavedFontSizeUsedAtStartup(t *testing.T) {
	services := newTestServices(t)
	st := shellState{}
	st.ensureSettings(services)
	st.settings.Interface.FontSize = 18
	st.saveSettings(services)

	var live float32
	tt := ui.NewTester(func(c *ui.Context) {
		App(c, services)
		live = theme.TokensOf(c).FontSize
	}, 1200, 800)
	tt.Frame()
	if live != 18 {
		t.Fatalf("startup font size = %v, want 18", live)
	}
}

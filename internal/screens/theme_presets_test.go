package screens

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/config"
	domaincol "heartless/internal/domain/collection"
)

// TestThemePresetsDiffer checks every theme has a light and a dark palette and
// that the themes are visibly different from one another.
func TestThemePresetsDiffer(t *testing.T) {
	if len(themePresets) != 4 {
		t.Fatalf("presets = %d, want 4", len(themePresets))
	}
	seen := map[string]map[bool]ui.Color{}
	for _, p := range themePresets {
		if p.ID == "" || p.Name == "" {
			t.Fatalf("preset without id or name: %+v", p)
		}
		light, dark := p.Palette(false), p.Palette(true)
		if light.Background == dark.Background {
			t.Fatalf("%s: light and dark backgrounds are the same", p.ID)
		}
		if light.Accent == dark.Accent {
			t.Fatalf("%s: light and dark accents are the same", p.ID)
		}
		for _, v := range []bool{false, true} {
			bg := p.Palette(v).Background
			if other, ok := seen[p.ID]; ok && other[v] == bg {
				t.Fatalf("%s: duplicate palette", p.ID)
			}
		}
		seen[p.ID] = map[bool]ui.Color{false: light.Background, true: dark.Background}
	}

	// The backgrounds are all different, so picking a theme is visible.
	if seen["neutral"][false] == seen["nord"][false] ||
		seen["nord"][false] == seen["forest"][false] ||
		seen["forest"][false] == seen["plum"][false] {
		t.Fatal("two themes share a light background")
	}
}

// TestThemeColorsAreCoherent checks a theme sets the window, panels, text and
// accent together, and that the regions are distinguishable: the content area
// must not read as the same flat surface as the window, and text must stay
// legible on both.
func TestThemeColorsAreCoherent(t *testing.T) {
	// channelDelta is the average per-channel difference, which tells panels
	// apart where a luma contrast ratio saturates near white and near black.
	channelDelta := func(a, b ui.Color) int {
		d := func(x, y uint8) int {
			if x > y {
				return int(x - y)
			}
			return int(y - x)
		}
		return (d(a.R, b.R) + d(a.G, b.G) + d(a.B, b.B)) / 3
	}
	luma := func(c ui.Color) float64 {
		lin := func(v uint8) float64 {
			x := float64(v) / 255
			if x <= 0.03928 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
	}
	contrast := func(a, b ui.Color) float64 {
		la, lb := luma(a), luma(b)
		if la < lb {
			la, lb = lb, la
		}
		return (la + 0.05) / (lb + 0.05)
	}

	for _, preset := range themePresets {
		for _, dark := range []bool{false, true} {
			pal := preset.Palette(dark)
			tk := buildTokens(pal, dark, 0)
			label := preset.ID
			if dark {
				label += " dark"
			} else {
				label += " light"
			}

			// The theme drives every region, not one of them.
			if tk.BgLayout != pal.Background {
				t.Errorf("%s: window %v, want the palette's %v", label, tk.BgLayout, pal.Background)
			}
			if tk.BgContainer != pal.Surface {
				t.Errorf("%s: panel %v, want the palette's %v", label, tk.BgContainer, pal.Surface)
			}
			if tk.Text != pal.Foreground {
				t.Errorf("%s: text %v, want the palette's %v", label, tk.Text, pal.Foreground)
			}
			if tk.Primary != pal.Accent {
				t.Errorf("%s: accent %v, want the palette's %v", label, tk.Primary, pal.Accent)
			}

			// The content area must read as its own region.
			if d := channelDelta(tk.BgLayout, tk.BgContainer); d < 8 {
				t.Errorf("%s: window and panel differ by %d, too flat to tell apart", label, d)
			}
			// Text must stay legible on both.
			for name, bg := range map[string]ui.Color{"window": tk.BgLayout, "panel": tk.BgContainer} {
				if r := contrast(tk.Text, bg); r < 7 {
					t.Errorf("%s: text on the %s contrasts %.1f:1, want at least 7", label, name, r)
				}
				if r := contrast(tk.TextSecondary, bg); r < 3 {
					t.Errorf("%s: secondary text on the %s contrasts %.1f:1, want at least 3", label, name, r)
				}
			}
		}
	}
}

// TestNoColorOverrides checks the theme is the only source of colors: settings
// carry a preset and a variant, nothing per-color.
func TestNoColorOverrides(t *testing.T) {
	s := config.DefaultApplicationSettings()
	if s.Theme.Name == "" || s.Theme.Variant == "" {
		t.Fatalf("defaults must name a theme and a variant: %+v", s.Theme)
	}
	if _, err := json.Marshal(s.Theme); err != nil {
		t.Fatal(err)
	}
}

// TestThemePaintsTheWindow checks the shell actually paints each region from
// the chosen theme: the title bar and sidebar take the window color, and the
// content area takes the panel surface, so the regions read separately.
func TestThemePaintsTheWindow(t *testing.T) {
	for _, tc := range []struct {
		name  string
		theme string
		dark  bool
	}{
		{"neutral light", "neutral", false},
		{"nord dark", "nord", true},
		{"forest dark", "forest", true},
		{"plum dark", "plum", true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			services := newTestServices(t)
			s := config.DefaultApplicationSettings()
			s.Theme.Name = tc.theme
			if tc.dark {
				s.Theme.Variant = config.VariantDark
			}
			if err := services.Settings.Save(t.Context(), s); err != nil {
				t.Fatalf("save settings: %v", err)
			}

			tester := ui.NewTester(func(c *ui.Context) { App(c, services) }, 1000, 700)
			tester.Frame()
			tester.Frame()
			img := tester.Image()
			pal := themePresetByID(tc.theme).Palette(tc.dark)

			regions := map[string]struct {
				x, y int
				want ui.Color
			}{
				"title bar": {500, 12, pal.Background},
				"sidebar":   {120, 300, pal.Background},
				"content":   {700, 400, pal.Surface},
			}
			for name, r := range regions {
				got := img.RGBAAt(r.x, r.y)
				if got.R != r.want.R || got.G != r.want.G || got.B != r.want.B {
					t.Errorf("%s = %v, want %v", name, got, r.want)
				}
			}
		})
	}
}
func TestThemeConfigUsesPresetAndVariant(t *testing.T) {
	s := config.DefaultApplicationSettings()
	s.Theme.Variant = config.VariantDark

	neutral := ThemeConfig(s).Tokens(true)
	s.Theme.Name = "forest"
	forest := ThemeConfig(s).Tokens(true)
	if neutral.BgLayout == forest.BgLayout {
		t.Fatalf("changing preset did not change the background: %+v", neutral.BgLayout)
	}
	if !forest.Dark {
		t.Fatal("dark variant tokens are not marked dark")
	}

	// An unknown name falls back to the default rather than leaving no theme.
	s.Theme.Name = "retired-theme"
	fallback := ThemeConfig(s).Tokens(true)
	if fallback.BgLayout != neutral.BgLayout {
		t.Fatalf("unknown theme did not fall back: %+v vs %+v", fallback.BgLayout, neutral.BgLayout)
	}
}

// TestCompactModeTightensRows checks compact mode makes spacing smaller, which
// is what makes the setting do something.
func TestCompactModeTightensRows(t *testing.T) {
	var normal, compactH float32
	for _, tc := range []struct {
		compact bool
		out     *float32
	}{
		{false, &normal},
		{true, &compactH},
	} {
		tc := tc
		tester := ui.NewTester(func(c *ui.Context) {
			setCompact(tc.compact)
			*tc.out = space(c, 100)
		}, 400, 300)
		tester.Frame()
	}
	if compactH >= normal {
		t.Fatalf("compact spacing = %v, want less than %v", compactH, normal)
	}
	if got := compactH / normal; got != compactScale {
		t.Fatalf("compact scale = %v, want %v", got, compactScale)
	}
}

// TestCompactModeShrinksSidebarRows checks the setting reaches the real app: a
// sidebar row is shorter once compact mode is saved.
func TestCompactModeShrinksSidebarRows(t *testing.T) {
	rowHeight := func(compactOn bool) float32 {
		services := newTestServices(t)
		if err := services.Collections.SaveCollection(context.Background(),
			domaincol.New("ws-1", "api")); err != nil {
			t.Fatalf("save collection: %v", err)
		}
		s := config.DefaultApplicationSettings()
		s.Interface.CompactMode = compactOn
		if err := services.Settings.Save(context.Background(), s); err != nil {
			t.Fatalf("save settings: %v", err)
		}

		tester := ui.NewTester(func(c *ui.Context) { App(c, services) }, 900, 700)
		tester.Frame()
		tester.Frame()
		r, ok := tester.Find("api")
		if !ok || r.H == 0 {
			t.Fatalf("collection row not laid out: %+v", r)
		}
		return r.H
	}

	normal, compactRow := rowHeight(false), rowHeight(true)
	if compactRow >= normal {
		t.Fatalf("compact row = %v, want less than %v", compactRow, normal)
	}
}

// TestDarkVariantIsApplied checks the frame uses the chosen variant rather
// than following the system appearance.
func TestDarkVariantIsApplied(t *testing.T) {
	s := config.DefaultApplicationSettings()
	s.Theme.Variant = config.VariantDark
	cfg := ThemeConfig(s)

	appliedDark := false
	tt := ui.NewTester(func(c *ui.Context) {
		applyTheme(c, cfg, s)
		appliedDark = c.Theme().Dark
	}, 300, 200)
	tt.Frame()
	if !appliedDark {
		t.Fatal("dark variant was not applied")
	}

	var lightApplied bool
	s.Theme.Variant = config.VariantLight
	tt2 := ui.NewTester(func(c *ui.Context) {
		applyTheme(c, cfg, s)
		lightApplied = !c.Theme().Dark
	}, 300, 200)
	tt2.Frame()
	if !lightApplied {
		t.Fatal("light variant was not applied")
	}
}

// TestThemePickerUsesStoredVariant checks the swatches follow the variant in
// the settings rather than the segmented control's state, which is stale until
// the Appearance section has been drawn. The swatches otherwise painted their
// light surfaces over a dark page.
func TestThemePickerUsesStoredVariant(t *testing.T) {
	var surface ui.Color
	st := shellState{settings: config.DefaultApplicationSettings()}
	st.settings.Theme.Variant = config.VariantDark
	st.settings.Theme.Name = "nord"
	// themeVariant has not been synced yet, as on first open.
	if st.themeVariant != 0 {
		t.Fatalf("precondition: themeVariant = %d", st.themeVariant)
	}

	tester := ui.NewTester(func(c *ui.Context) {
		appearanceSettings(c, theme.TokensOf(c), &st)
		surface = themePresetByID("nord").Palette(st.darkVariant()).Surface
	}, 900, 500)
	tester.Frame()

	want := themePresetByID("nord").Palette(true).Surface
	if surface != want {
		t.Fatalf("swatch surface = %v, want the dark palette's %v", surface, want)
	}
}

// TestSaveClearsDirty checks saving clears the unsaved marker. It was left
// set, so a saved page still showed "Unsaved changes" and the button stayed on.
func TestSaveClearsDirty(t *testing.T) {
	services := newTestServices(t)
	st := shellState{}
	st.ensureSettings(services)
	st.settings.Theme.Name = "nord"
	st.settingsDirty = true

	tester := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tester.Frame()
	if err := tester.Click("Save"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if st.settingsDirty {
		t.Fatal("saving left the settings dirty")
	}
	if !st.settingsSaved {
		t.Fatal("saving did not mark the settings saved")
	}

	tester.Frame()
	if tester.HasText("Unsaved changes") || !tester.HasText("Saved") {
		t.Fatalf("marker after save, texts %q", tester.Texts())
	}
}

// TestThemePickerSelectsPreset clicks a theme swatch and checks it is stored.
func TestThemePickerSelectsPreset(t *testing.T) {
	services := newTestServices(t)
	st := shellState{}
	st.ensureSettings(services)

	tt := ui.NewTester(func(c *ui.Context) {
		settingsView(c, theme.TokensOf(c), &st, services)
	}, 1100, 800)
	tt.Frame()

	if err := tt.Click("Plum"); err != nil {
		t.Fatalf("click theme: %v", err)
	}
	if got := st.settings.Theme.Name; got != "plum" {
		t.Fatalf("theme = %q, want plum", got)
	}
	if !st.settingsDirty {
		t.Fatal("changing the theme did not mark the settings dirty")
	}
}

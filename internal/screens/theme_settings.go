package screens

import (
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/config"
)

// ThemeConfig builds a Lobe MyGO theme Config from the saved settings: the
// chosen theme preset in both appearances, the user's own colors applied over
// it, and the interface font size scaled.
//
// Keep the returned Config for the app's life: every Config is remembered, and
// TokensOf looks it up by the theme the frame uses.
func ThemeConfig(s config.ApplicationSettings) *theme.Config {
	preset := themePresetByID(s.Theme.Name)
	light := buildTokens(preset.Palette(false), false, s.Interface.FontSize)
	dark := buildTokens(preset.Palette(true), true, s.Interface.FontSize)
	return theme.NewConfig(light, dark)
}

// applyVariant applies the theme the settings chose, light or dark, instead of
// following the system's appearance, so the Variant control means what it says.
func applyTheme(c *ui.Context, cfg *theme.Config, s config.ApplicationSettings) {
	c.SetTheme(cfg.Theme(ThemeIsDark(s)))
}

// applyFontSize scales a token set's text from the interface font size, which
// zero leaves at the library's default.
func applyFontSize(t *theme.Tokens, size int) {
	if size <= 0 {
		return
	}
	scale := float32(size) / t.FontSize
	t.FontSize = float32(size)
	t.FontSizeSM = t.FontSizeSM * scale
	t.FontSizeLG = t.FontSizeLG * scale
}

// baseFontSize is the interface font size the hand-tuned sizes in the screens
// were laid out at: textSize's sizes are relative to it.
const baseFontSize float32 = 14

// textSize scales a text size tuned against baseFontSize to the frame's font
// size, so the interface font size setting resizes labels, icons and code
// alike. The theme's font size is the one chosen in settings.
func textSize(c *ui.Context, size float32) float32 {
	return c.Theme().Rem(size / baseFontSize)
}

// compactScale is how much compact mode shrinks spacing: row heights, gaps and
// padding, but not text, which has its own setting.
const compactScale float32 = 0.78

// compactLayout is whether the frame being built is compact. One frame builds
// at a time, so a single flag carries the density into helpers deep in the
// widget tree without threading it through every signature.
var compactLayout atomic.Bool

// setCompact records the frame's density for space to read.
func setCompact(compact bool) { compactLayout.Store(compact) }

// space scales a spacing value (a height, gap or padding) for the frame's
// density, so compact mode tightens the layout everywhere it is applied.
func space(c *ui.Context, v float32) float32 {
	_ = c
	if compactLayout.Load() {
		return v * compactScale
	}
	return v
}

// ThemeIsDark reports whether the saved theme asks for the dark variant.
func ThemeIsDark(s config.ApplicationSettings) bool {
	return s.Theme.Variant == config.VariantDark
}

// parseHexColor parses "#rgb" or "#rrggbb".
func parseHexColor(s string) (ui.Color, bool) {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	hex := func(b byte) (uint8, bool) {
		switch {
		case b >= '0' && b <= '9':
			return b - '0', true
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10, true
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10, true
		}
		return 0, false
	}
	switch len(s) {
	case 3:
		var v [3]uint8
		for i := 0; i < 3; i++ {
			h, ok := hex(s[i])
			if !ok {
				return ui.Color{}, false
			}
			v[i] = h*16 + h
		}
		return ui.RGBA(v[0], v[1], v[2], 255), true
	case 6:
		var v [3]uint8
		for i := 0; i < 3; i++ {
			hi, ok1 := hex(s[i*2])
			lo, ok2 := hex(s[i*2+1])
			if !ok1 || !ok2 {
				return ui.Color{}, false
			}
			v[i] = hi*16 + lo
		}
		return ui.RGBA(v[0], v[1], v[2], 255), true
	}
	return ui.Color{}, false
}

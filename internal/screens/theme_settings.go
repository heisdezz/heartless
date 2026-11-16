package screens

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/config"
)

// ThemeConfig builds a Lobe MyGO theme Config from the saved settings. The
// light and dark token sets are the library defaults, with the accent and
// background colors overridden when the user set them, and the interface
// font size scaled.
//
// Keep the returned Config for the app's life: every Config is remembered, and
// TokensOf looks it up by the theme the frame uses.
func ThemeConfig(s config.ApplicationSettings) *theme.Config {
	light := *theme.Default().Tokens(false)
	dark := *theme.Default().Tokens(true)

	applyThemeSettings(&light, s.Theme, false)
	applyThemeSettings(&dark, s.Theme, true)
	applyFontSize(&light, s.Interface.FontSize)
	applyFontSize(&dark, s.Interface.FontSize)
	return theme.NewConfig(light, dark)
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

// ThemeIsDark reports whether the saved theme asks for the dark variant.
func ThemeIsDark(s config.ApplicationSettings) bool {
	return s.Theme.Name == "dark"
}

// applyThemeSettings overrides a token set from the theme settings.
func applyThemeSettings(t *theme.Tokens, cfg config.Theme, dark bool) {
	if cfg.AccentColor != "" {
		if c, ok := parseHexColor(cfg.AccentColor); ok {
			t.Primary = c
			t.PrimaryText = c
		}
	}
	// The saved background/foreground describe one appearance; apply them to the
	// variant they were picked for.
	if cfg.BackgroundColor != "" && dark == ThemeIsDarkConfig(cfg) {
		if c, ok := parseHexColor(cfg.BackgroundColor); ok {
			t.BgLayout = c
		}
	}
	if cfg.ForegroundColor != "" && dark == ThemeIsDarkConfig(cfg) {
		if c, ok := parseHexColor(cfg.ForegroundColor); ok {
			t.Text = c
		}
	}
}

func ThemeIsDarkConfig(cfg config.Theme) bool { return cfg.Name == "dark" }

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

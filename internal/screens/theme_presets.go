package screens

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// ThemePreset is one of the app's built-in themes. Each has a light and a
// dark variant; Palette recolors a token set for it.
type ThemePreset struct {
	// ID is stored in the settings and must stay stable.
	ID string
	// Name is what the settings show.
	Name string
	// Palette returns the colors a variant of the theme is built from.
	Palette func(dark bool) Palette
}

// Palette is the handful of colors a theme sets. The rest of the token set is
// derived from these, so a theme is a set of brand colors rather than a
// hundred hand-picked values.
type Palette struct {
	// Accent is the primary color: buttons, links, selection.
	Accent ui.Color
	// Background is the window background, the palette's tint included.
	Background ui.Color
	// Surface is the raised background of cards, panels and inputs.
	Surface ui.Color
	// Foreground is the body text color.
	Foreground ui.Color
	// Border is the hairline between regions.
	Border ui.Color
}

// themePresets lists the built-in themes in the order the settings show them.
var themePresets = []ThemePreset{
	{ID: "neutral", Name: "Neutral", Palette: neutralPalette},
	{ID: "nord", Name: "Nord", Palette: nordPalette},
	{ID: "forest", Name: "Forest", Palette: forestPalette},
	{ID: "plum", Name: "Plum", Palette: plumPalette},
}

// defaultThemeID is the theme new settings start with.
const defaultThemeID = "neutral"

// themePresetByID returns the preset with id, falling back to the default so an
// unknown or retired name still themes the app.
func themePresetByID(id string) ThemePreset {
	for _, p := range themePresets {
		if p.ID == id {
			return p
		}
	}
	return themePresets[0]
}

// themePresetIndex returns the position of id in themePresets, or the default's.
func themePresetIndex(id string) int {
	for i, p := range themePresets {
		if p.ID == id {
			return i
		}
	}
	return 0
}

func neutralPalette(dark bool) Palette {
	if dark {
		return Palette{
			Accent:     ui.RGBA(0x3b, 0x82, 0xf6, 1),
			Background: ui.RGBA(0x18, 0x18, 0x1b, 1),
			Surface:    ui.RGBA(0x26, 0x26, 0x2b, 1),
			Foreground: ui.RGBA(0xe6, 0xe6, 0xea, 1),
			Border:     ui.RGBA(0x33, 0x33, 0x39, 1),
		}
	}
	return Palette{
		Accent:     ui.RGBA(0x00, 0x66, 0xff, 1),
		Background: ui.RGBA(0xff, 0xff, 0xff, 1),
		Surface:    ui.RGBA(0xf1, 0xf2, 0xf5, 1),
		Foreground: ui.RGBA(0x11, 0x11, 0x14, 1),
		Border:     ui.RGBA(0xe3, 0xe3, 0xe8, 1),
	}
}

// nordPalette is the Nord scheme's cool blue-grey.
func nordPalette(dark bool) Palette {
	if dark {
		return Palette{
			Accent:     ui.RGBA(0x88, 0xc0, 0xd0, 1),
			Background: ui.RGBA(0x2e, 0x34, 0x40, 1), // nord0
			Surface:    ui.RGBA(0x3b, 0x42, 0x52, 1), // nord1
			Foreground: ui.RGBA(0xec, 0xef, 0xf4, 1), // nord6
			Border:     ui.RGBA(0x4c, 0x56, 0x6a, 1), // nord2
		}
	}
	return Palette{
		Accent:     ui.RGBA(0x5e, 0x81, 0xac, 1), // nord10
		Background: ui.RGBA(0xec, 0xef, 0xf4, 1), // nord6
		Surface:    ui.RGBA(0xde, 0xe4, 0xed, 1), // nord5
		Foreground: ui.RGBA(0x2e, 0x34, 0x40, 1), // nord0
		Border:     ui.RGBA(0xd8, 0xde, 0xe9, 1), // nord4
	}
}

// forestPalette is green, after the app's name.
func forestPalette(dark bool) Palette {
	if dark {
		return Palette{
			Accent:     ui.RGBA(0x4a, 0xde, 0x80, 1),
			Background: ui.RGBA(0x0f, 0x1a, 0x14, 1),
			Surface:    ui.RGBA(0x1d, 0x30, 0x25, 1),
			Foreground: ui.RGBA(0xdd, 0xf0, 0xe3, 1),
			Border:     ui.RGBA(0x25, 0x3a, 0x2e, 1),
		}
	}
	return Palette{
		Accent:     ui.RGBA(0x16, 0x7a, 0x3f, 1),
		Background: ui.RGBA(0xfb, 0xfd, 0xfb, 1),
		Surface:    ui.RGBA(0xe8, 0xf1, 0xea, 1),
		Foreground: ui.RGBA(0x14, 0x1f, 0x18, 1),
		Border:     ui.RGBA(0xd9, 0xe7, 0xdd, 1),
	}
}

// plumPalette is warm purple.
func plumPalette(dark bool) Palette {
	if dark {
		return Palette{
			Accent:     ui.RGBA(0xc0, 0x8a, 0xff, 1),
			Background: ui.RGBA(0x1a, 0x14, 0x20, 1),
			Surface:    ui.RGBA(0x2d, 0x24, 0x38, 1),
			Foreground: ui.RGBA(0xea, 0xe2, 0xf2, 1),
			Border:     ui.RGBA(0x36, 0x2b, 0x42, 1),
		}
	}
	return Palette{
		Accent:     ui.RGBA(0x76, 0x2c, 0xb5, 1),
		Background: ui.RGBA(0xfd, 0xfb, 0xff, 1),
		Surface:    ui.RGBA(0xef, 0xe7, 0xf7, 1),
		Foreground: ui.RGBA(0x21, 0x18, 0x2b, 1),
		Border:     ui.RGBA(0xe6, 0xdd, 0xf0, 1),
	}
}

// buildTokens makes a token set for a theme variant: the library's defaults
// for that appearance, recolorized entirely from the palette.
func buildTokens(p Palette, dark bool, fontSize int) theme.Tokens {
	t := *theme.Default().Tokens(dark)

	// Text and its variants.
	t.Text = p.Foreground
	t.TextSecondary = mix(p.Foreground, p.Background, 0.28)
	t.TextTertiary = mix(p.Foreground, p.Background, 0.48)
	t.TextQuaternary = mix(p.Foreground, p.Background, 0.62)
	t.TextPlaceholder = t.TextTertiary

	// Backgrounds. The window and the panels a step apart from it give the app
	// its regions: the sidebar and the title bar sit at the window's color, and
	// the content area, cards and fields on the surface one. The separation is
	// deliberately more than a hair, or the regions read as one flat sheet.
	t.BgLayout = p.Background
	t.BgBase = p.Background
	t.BgContainer = p.Surface
	t.BgElevated = mix(p.Surface, p.Foreground, 0.06)
	t.BgContainerSecondary = mix(p.Surface, p.Background, 0.5)
	t.BgContainerDisabled = mix(p.Background, p.Foreground, 0.02)
	t.BgSpotlight = mix(p.Surface, p.Foreground, 0.04)

	// Fills, from the faintest wash to a solid tint.
	t.FillQuaternary = mix(p.Background, p.Foreground, 0.04)
	t.FillTertiary = mix(p.Background, p.Foreground, 0.07)
	t.FillSecondary = mix(p.Background, p.Foreground, 0.10)
	t.Fill = mix(p.Background, p.Foreground, 0.14)

	// Borders.
	t.Border = mix(p.Border, p.Foreground, 0.12)
	t.BorderSecondary = p.Border
	t.BorderBg = p.Border

	// Accent, and the washes built from it.
	t.Primary = p.Accent
	t.PrimaryText = p.Accent
	t.PrimaryBorder = mix(p.Accent, p.Background, 0.35)
	t.PrimaryBg = fade(p.Accent, 0.22)
	t.PrimaryBgHover = fade(p.Accent, 0.32)
	t.PrimaryFill = fade(p.Accent, 0.24)
	t.PrimaryFillSecondary = fade(p.Accent, 0.16)
	t.PrimaryFillTertiary = fade(p.Accent, 0.10)
	t.PrimaryFillQuaternary = fade(p.Accent, 0.06)

	// Selection follows the accent, so highlighting matches the theme.
	t.PrimaryBorderHover = p.Accent

	// Status colors, deepened on light backgrounds so they stay readable.
	if dark {
		t.Success = ui.RGBA(0x4a, 0xde, 0x80, 1)
		t.Warning = ui.RGBA(0xf5, 0xc2, 0x42, 1)
		t.Error = ui.RGBA(0xf8, 0x71, 0x71, 1)
		t.Info = ui.RGBA(0x60, 0xa5, 0xfa, 1)
	} else {
		t.Success = ui.RGBA(0x12, 0x84, 0x3d, 1)
		t.Warning = ui.RGBA(0xb4, 0x53, 0x09, 1)
		t.Error = ui.RGBA(0xdc, 0x26, 0x26, 1)
		t.Info = ui.RGBA(0x1d, 0x4e, 0xd8, 1)
	}
	t.SuccessBg = fade(t.Success, 0.16)
	t.ErrorBg = fade(t.Error, 0.16)
	t.WarningBg = fade(t.Warning, 0.16)
	t.InfoBg = fade(t.Info, 0.16)

	applyFontSize(&t, fontSize)
	return t
}

// mix blends b over a by amount, the way a theme background carries a slight
// tint of its foreground.
func mix(a, b ui.Color, amount float32) ui.Color {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	blend := func(x, y uint8) uint8 {
		return uint8(float32(x)*(1-amount) + float32(y)*amount + 0.5)
	}
	return ui.RGBA(blend(a.R, b.R), blend(a.G, b.G), blend(a.B, b.B), 1)
}

// fade returns c at the given opacity, for a wash drawn over a surface.
func fade(c ui.Color, alpha float32) ui.Color {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return ui.RGBA(c.R, c.G, c.B, alpha)
}

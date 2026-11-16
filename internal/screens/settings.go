package screens

import (
	"context"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	"heartless/internal/config"
)

// settingsSection selects which settings panel shows.
type settingsSection int

const (
	sectionAppearance settingsSection = iota
	sectionInterface
	sectionPlugins
)

// settingsSections names the panels, in the order the sidebar lists them.
var settingsSections = []string{"Appearance", "Interface", "Plugins"}

// minimumFontSize and maximumFontSize bound the interface font size setting.
const (
	minimumFontSize = 10
	maximumFontSize = 22
)

// settingsView is the settings page: a section list on the left and the chosen
// section's form on the right, with Save applying the changes.
func settingsView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	st.ensureSettings(services)

	ui.Column(c).Fill().MinWidth(0).Children(func() {
		pageHeader(c, tk, "Settings", "Appearance, interface behavior and plugins.")

		// Save bar.
		ui.Row(c).Height(space(c, 44)).PaddingX(24).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			save := action.Button(c, "Save", action.ButtonProps{Variant: action.Primary, Size: theme.Small}).Shrink(0)
			if !st.settingsDirty {
				save.Disabled(true)
			}
			if save.Clicked() && st.settingsDirty {
				st.saveSettings(services)
			}
			switch {
			case st.settingsDirty:
				ui.Text(c, "Unsaved changes").FontSize(textSize(c, 12)).TextColor(tk.TextTertiary).SingleLine()
			case st.settingsSaved:
				ui.Text(c, "Saved").FontSize(textSize(c, 12)).TextColor(tk.Success).SingleLine()
			}
			ui.Box(c).Grow(1)
			action.Button(c, "Reset to defaults", action.ButtonProps{Variant: action.Text, Size: theme.Small}).
				Shrink(0).OnClick(func() { st.resetSettings(services) })
		})

		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
			// Section list.
			ui.Column(c).Width(220).Shrink(0).Padding(12, 12).Gap(2).
				AlignItems(ui.Stretch).BorderWidth(0, 1, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
				for i, name := range settingsSections {
					section := settingsSection(i)
					btn := action.Button(c, name, action.ButtonProps{Variant: action.Text}).
						Justify(ui.Start).AlignSelf(ui.Stretch)
					if st.settingsSection == section {
						btn.Background(tk.FillSecondary).FontWeight(600).TextColor(tk.Text)
					} else {
						btn.TextColor(tk.TextSecondary)
					}
					btn.OnClick(func() { st.settingsSection = section })
				}
			})

			ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				ui.Column(c).AlignSelf(ui.Stretch).Grow(1).MinWidth(0).Padding(24).Children(func() {
					// A reading-width column keeps rows from stretching across a wide
					// window, and keeps the controls near their labels.
					ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(720).Gap(16).Children(func() {
						switch st.settingsSection {
						case sectionAppearance:
							appearanceSettings(c, tk, st)
						case sectionInterface:
							interfaceSettings(c, tk, st)
						case sectionPlugins:
							pluginSettings(c, tk, st)
						}
					})
				})
			})
		})
	})
}

// appearanceSettings picks the theme. The theme owns every color the
// interface draws with: window, panels, text and accent. Nothing is set
// piecemeal, so the four presets are the whole palette.
func appearanceSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	settingsGroup(c, tk, "Theme", "How the interface looks.")(func() {
		rowSetting(c, tk, "Theme", "Each theme sets the window, panels, text and accent.")(func() {
			themePicker(c, tk, st)
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Variant", "Light or dark appearance.")(func() {
			// The control shows the stored variant; the settings are the truth.
			st.themeVariant = 0
			if st.darkVariant() {
				st.themeVariant = 1
			}
			action.Segmented(c, &st.themeVariant, []string{"Light", "Dark"}, action.SegmentedProps{}).
				Width(160).Shrink(0).OnChange(func() { st.syncThemeVariant() })
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Colors", "What the theme sets, for the current variant.")(func() {
			themeColorPreview(c, tk, st)
		})
	})
}

// themeColorPreview shows the colors the chosen theme applies, so what it
// controls is visible without being editable.
func themeColorPreview(c *ui.Context, tk *theme.Tokens, st *shellState) {
	p := themePresetByID(st.settings.Theme.Name).Palette(st.darkVariant())
	colors := []struct {
		name string
		col  ui.Color
	}{
		{"Window", p.Background},
		{"Panel", p.Surface},
		{"Text", p.Foreground},
		{"Border", p.Border},
		{"Accent", p.Accent},
	}
	ui.Row(c).Gap(12).Shrink(0).Children(func() {
		for _, c2 := range colors {
			c2 := c2
			ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Width(space(c, 22)).Height(space(c, 22)).Radius(tk.BorderRadiusSM).
					Background(c2.col).Border(1, tk.BorderSecondary)
				ui.Text(c, c2.name).FontSize(textSize(c, 10)).TextColor(tk.TextTertiary)
			})
		}
	})
}

// themePicker shows the built-in themes as swatches of the chosen variant,
// the selected one outlined in its accent.
func themePicker(c *ui.Context, tk *theme.Tokens, st *shellState) {
	dark := st.darkVariant()
	ui.Row(c).Gap(8).Shrink(0).Children(func() {
		for _, preset := range themePresets {
			preset := preset
			selected := st.settings.Theme.Name == preset.ID
			p := preset.Palette(dark)
			btn := ui.ButtonBase(c).Label(preset.Name).Width(space(c, 92)).Height(space(c, 56)).
				Radius(tk.BorderRadiusSM).Cursor(ui.CursorPointer).Transition(rowTransition)
			if selected {
				btn.Border(2, p.Accent).Background(p.Surface)
			} else {
				btn.Border(1, tk.BorderSecondary).Background(p.Surface)
			}
			btn.Children(func() {
				ui.Row(c).Gap(4).Children(func() {
					ui.Box(c).Width(space(c, 10)).Height(space(c, 10)).Radius(5).Background(p.Accent)
					ui.Box(c).Width(space(c, 10)).Height(space(c, 10)).Radius(5).Background(p.Foreground)
					ui.Box(c).Width(space(c, 10)).Height(space(c, 10)).Radius(5).Background(p.Border)
				})
				ui.Text(c, preset.Name).FontSize(textSize(c, 11)).TextColor(p.Foreground)
			})
			if btn.Clicked() && !selected {
				st.settings.Theme.Name = preset.ID
				st.syncThemeVariant()
			}
		}
	})
}

// darkVariant reports whether the settings ask for the dark variant. It reads
// the settings rather than themeVariant, which only the segmented control
// updates and so is stale until the Appearance section has been drawn.
func (s *shellState) darkVariant() bool {
	return s.settings.Theme.Variant == config.VariantDark
}

// syncThemeVariant keeps the stored variant in step with the variant control.
func (s *shellState) syncThemeVariant() {
	if s.themeVariant == 1 {
		s.settings.Theme.Variant = config.VariantDark
	} else {
		s.settings.Theme.Variant = config.VariantLight
	}
	if s.settings.Theme.Name == "" {
		s.settings.Theme.Name = defaultThemeID
	}
	s.settingsDirty = modifiedFrom(s.settings, s.settingsBase)
}

// interfaceSettings edits the interface preferences.
func interfaceSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	ic := &st.settings.Interface

	settingsGroup(c, tk, "Interface", "Behavior and density preferences.")(func() {
		rowSetting(c, tk, "Reduce motion", "Fewer animations, for accessibility.")(func() {
			choice.Switch(c, &ic.ReduceMotion, choice.SwitchProps{}).Shrink(0)
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Compact mode", "Tighter spacing where available.")(func() {
			choice.Switch(c, &ic.CompactMode, choice.SwitchProps{}).Shrink(0)
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Show icons", "Icons alongside labels.")(func() {
			choice.Switch(c, &ic.ShowIcons, choice.SwitchProps{}).Shrink(0)
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Font size", "Size of the interface text, in points.")(func() {
			value := settingsFontSize(ic.FontSize)
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Width(44).Height(24).Shrink(0).Center().Radius(tk.BorderRadiusSM).
					Background(tk.FillSecondary).Children(func() {
					ui.Text(c, strconv.Itoa(int(math.Round(value)))).FontSize(textSize(c, 12)).FontWeight(600).SingleLine()
				})
				ui.Box(c).Width(200).Shrink(0).Clip().Children(func() {
					choice.Slider(c, &value, choice.SliderProps{
						Min: minimumFontSize, Max: maximumFontSize, Step: 1,
					}).OnChange(func() {
						ic.FontSize = int(math.Round(value))
						st.settingsDirty = modifiedFrom(st.settings, st.settingsBase)
					})
				})
			})
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Double-click action", "What a double-click on a request does.")(func() {
			ui.Box(c).Width(200).Shrink(0).Children(func() {
				field.Select(c, &ic.DoubleClickAction, field.SelectProps{
					Size: theme.Small,
					Options: []field.SelectOption{
						{Value: "open", Label: "Open"},
						{Value: "edit", Label: "Edit"},
						{Value: "none", Label: "None"},
					},
				})
			})
		})
	})
}

// settingsFontSize returns the font size the control edits: the saved one, or
// the theme's default when none is set.
func settingsFontSize(saved int) float64 {
	if saved <= 0 {
		return float64(math.Round(float64(theme.Default().Tokens(false).FontSize)))
	}
	return float64(saved)
}

// pluginSettings lists available and enabled plugins.
func pluginSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	pl := &st.settings.Plugins
	settingsGroup(c, tk, "Plugins", "Optional extensions bundled with Heartless.")(func() {
		if len(pl.Available) == 0 {
			ui.Text(c, "No plugins are available in this build.").FontSize(textSize(c, 13)).
				TextColor(tk.TextSecondary).PaddingY(4)
			return
		}
		for i, name := range pl.Available {
			if i > 0 {
				rowDivider(c, tk)
			}
			name := name
			rowSetting(c, tk, name, "")(func() {
				enabled := contains(pl.Enabled, name)
				choice.Switch(c, &enabled, choice.SwitchProps{}).Shrink(0).OnChange(func() {
					pl.Enabled = setMembership(pl.Enabled, name, enabled)
					st.settingsDirty = modifiedFrom(st.settings, st.settingsBase)
				})
			})
		}
	})
}

// settingsGroup renders a framed card holding a section's heading, caption and
// rows, so each group reads as one block.
func settingsGroup(c *ui.Context, tk *theme.Tokens, title, caption string) func(rows func()) {
	return func(rows func()) {
		ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(12).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, title).FontSize(textSize(c, 15)).FontWeight(600)
				if caption != "" {
					ui.Text(c, caption).FontSize(textSize(c, 12)).TextColor(tk.TextSecondary)
				}
			})
			ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Radius(tk.BorderRadius).
				Border(1, tk.BorderSecondary).PaddingX(16).Children(rows)
		})
	}
}

// rowDivider separates consecutive setting rows.
func rowDivider(c *ui.Context, tk *theme.Tokens) {
	ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
}

// rowSetting renders a labelled setting row with the control on the right.
// The hint may wrap rather than being cut off beside a wide control.
func rowSetting(c *ui.Context, tk *theme.Tokens, label, hint string) func(control func()) {
	return func(control func()) {
		ui.Row(c).AlignSelf(ui.Stretch).MinWidth(0).MinHeight(space(c, 52)).Gap(space(c, 24)).AlignItems(ui.Center).
			PaddingY(10).Children(func() {
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, label).FontSize(textSize(c, 13)).TextColor(tk.Text)
				if hint != "" {
					ui.Text(c, hint).FontSize(textSize(c, 11)).TextColor(tk.TextTertiary)
				}
			})
			ui.Box(c).Shrink(0).Children(control)
		})
	}
}

// ensureSettings loads the settings once and seeds the editor fields.
func (s *shellState) ensureSettings(services *app.Services) {
	if s.settingsLoaded || services == nil || services.Settings == nil {
		return
	}
	s.settingsLoaded = true
	loaded, err := services.Settings.Load(context.Background())
	if err != nil {
		s.err = err.Error()
		return
	}
	s.settings = loaded
	s.settingsBase = loaded
	s.themeVariant = 0
	if loaded.Theme.Variant == config.VariantDark {
		s.themeVariant = 1
	}
}

// saveSettings persists the settings and applies the theme.
func (s *shellState) saveSettings(services *app.Services) {
	if services == nil || services.Settings == nil {
		return
	}
	if err := services.Settings.Save(context.Background(), s.settings); err != nil {
		s.err = err.Error()
		return
	}
	s.settingsBase = s.settings
	s.settingsDirty = false
	s.settingsSaved = true
	// App rebuilds the theme from the saved copy on the next frame.
}

// resetSettings restores the defaults into the editor and saves them.
func (s *shellState) resetSettings(services *app.Services) {
	s.settings = config.DefaultApplicationSettings()
	s.themeVariant = 0
	s.settingsSaved = false
	s.saveSettings(services)
}

// contains reports whether list has v.
func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// setMembership adds or removes v from list and returns it.
func setMembership(list []string, v string, in bool) []string {
	if in {
		if contains(list, v) {
			return list
		}
		return append(list, v)
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if item != v {
			out = append(out, item)
		}
	}
	return out
}

// modifiedFrom reports whether s differs from base.
func modifiedFrom(s, base config.ApplicationSettings) bool {
	if s.Theme.Name != base.Theme.Name ||
		s.Theme.Variant != base.Theme.Variant {
		return true
	}
	if s.Interface.ReduceMotion != base.Interface.ReduceMotion ||
		s.Interface.CompactMode != base.Interface.CompactMode ||
		s.Interface.ShowIcons != base.Interface.ShowIcons ||
		s.Interface.DoubleClickAction != base.Interface.DoubleClickAction ||
		s.Interface.FontSize != base.Interface.FontSize {
		return true
	}
	if len(s.Plugins.Enabled) != len(base.Plugins.Enabled) {
		return true
	}
	for _, name := range s.Plugins.Enabled {
		if !contains(base.Plugins.Enabled, name) {
			return true
		}
	}
	return false
}

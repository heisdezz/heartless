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
		ui.Row(c).Height(44).PaddingX(24).Gap(8).AlignItems(ui.Center).
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

// appearanceSettings edits the theme: variant, accent and colors.
func appearanceSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	th := &st.settings.Theme

	// The stored name follows the segmented control, also for a change made
	// after this row built.
	st.syncThemeVariant()

	settingsGroup(c, tk, "Theme", "How the interface looks.")(func() {
		rowSetting(c, tk, "Variant", "Light or dark appearance.")(func() {
			action.Segmented(c, &st.themeVariant, []string{"Light", "Dark"}, action.SegmentedProps{}).
				Width(160).Shrink(0).OnChange(func() { st.syncThemeVariant() })
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Accent color", "Primary color for buttons and highlights.")(func() {
			colorField(c, tk, st, &th.AccentColor, "#0066ff")
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Background", "Main window background. Applies to the current variant.")(func() {
			colorField(c, tk, st, &th.BackgroundColor, "#ffffff")
		})
		rowDivider(c, tk)
		rowSetting(c, tk, "Foreground", "Main text color. Applies to the current variant.")(func() {
			colorField(c, tk, st, &th.ForegroundColor, "#000000")
		})
	})
}

// syncThemeVariant keeps the stored theme name in step with the variant
// control.
func (s *shellState) syncThemeVariant() {
	if s.themeVariant == 1 {
		s.settings.Theme.Name = "dark"
	} else {
		s.settings.Theme.Name = "light"
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
		ui.Row(c).AlignSelf(ui.Stretch).MinWidth(0).MinHeight(52).Gap(24).AlignItems(ui.Center).
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

// colorField renders a hex color input with a swatch preview. A valid value
// marks the settings dirty; an invalid one only shows its fallback swatch
// until it parses.
func colorField(c *ui.Context, tk *theme.Tokens, st *shellState, value *string, placeholder string) {
	ui.Row(c).Width(220).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		swatch := tk.BorderSecondary
		if col, ok := parseHexColor(*value); ok {
			swatch = col
		}
		ui.Box(c).Width(20).Height(20).Radius(tk.BorderRadiusSM).Background(swatch).
			Border(1, tk.BorderSecondary).Shrink(0)
		ui.Box(c).Grow(1).Basis(0).MinWidth(0).Clip().Children(func() {
			field.Input(c, value, field.InputProps{Placeholder: placeholder, Size: theme.Small}).Grow(1).
				OnChange(func() { st.settingsDirty = modifiedFrom(st.settings, st.settingsBase) })
		})
	})
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
	if loaded.Theme.Name == "dark" {
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
		s.Theme.AccentColor != base.Theme.AccentColor ||
		s.Theme.BackgroundColor != base.Theme.BackgroundColor ||
		s.Theme.ForegroundColor != base.Theme.ForegroundColor {
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

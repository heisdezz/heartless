package screens

import (
	"context"

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

// settingsView is the settings page: a section list on the left and the chosen
// section's form on the right, with Save applying the changes.
func settingsView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	st.ensureSettings(services)

	ui.Column(c).Fill().MinWidth(0).Children(func() {
		pageHeader(c, tk, "Settings", "Appearance, interface behavior and plugins.")

		// Save bar.
		ui.Row(c).Height(44).PaddingX(24).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			if action.Button(c, "Save", action.ButtonProps{Variant: action.Primary}).Clicked() {
				st.saveSettings(services)
			}
			if st.settingsDirty {
				ui.Text(c, "Unsaved changes").FontSize(12).TextColor(tk.TextTertiary)
			} else if st.settingsSaved {
				ui.Text(c, "Saved").FontSize(12).TextColor(tk.Success)
			}
			ui.Box(c).Grow(1)
			if action.Button(c, "Reset to defaults", action.ButtonProps{Variant: action.Text}).Clicked() {
				st.settings = config.DefaultApplicationSettings()
				st.settingsDirty = true
				st.settingsSaved = false
			}
		})

		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
			// Section list.
			ui.Column(c).Width(200).Shrink(0).Padding(12, 12).Gap(2).
				BorderWidth(0, 1, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
				for i, name := range []string{"Appearance", "Interface", "Plugins"} {
					i, name := settingsSection(i), name
					btn := action.Button(c, name, action.ButtonProps{Variant: action.Text})
					if st.settingsSection == i {
						btn.Background(tk.FillSecondary).FontWeight(600).TextColor(tk.Text)
					} else {
						btn.TextColor(tk.TextSecondary)
					}
					btn.AlignSelf(ui.Stretch)
					if btn.Clicked() {
						st.settingsSection = i
					}
				}
			})

			ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				ui.Column(c).AlignSelf(ui.Stretch).Grow(1).MinWidth(0).Padding(24).Gap(16).Children(func() {
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
}

// appearanceSettings edits the theme: variant, accent and colors.
func appearanceSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	th := &st.settings.Theme

	settingsGroup(c, tk, "Theme", "How the interface looks.")
	rowSetting(c, tk, "Variant", "Light or dark appearance.")(func() {
		action.Segmented(c, &st.themeVariant, []string{"Light", "Dark"}, action.SegmentedProps{}).Width(160)
	})
	// Keep the stored name in step with the segmented control.
	name := "light"
	if st.themeVariant == 1 {
		name = "dark"
	}
	th.Name = name

	rowSetting(c, tk, "Accent color", "Primary color for buttons and highlights.")(func() {
		colorField(c, tk, &th.AccentColor, "#0066ff")
	})
	rowSetting(c, tk, "Background", "Main window background.")(func() {
		colorField(c, tk, &th.BackgroundColor, "#ffffff")
	})
	rowSetting(c, tk, "Foreground", "Main text color.")(func() {
		colorField(c, tk, &th.ForegroundColor, "#000000")
	})
}

// interfaceSettings edits the interface preferences.
func interfaceSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	ic := &st.settings.Interface

	settingsGroup(c, tk, "Interface", "Behavior and density preferences.")
	rowSetting(c, tk, "Reduce motion", "Fewer animations, for accessibility.")(func() {
		choice.Switch(c, &ic.ReduceMotion, choice.SwitchProps{})
	})
	rowSetting(c, tk, "Compact mode", "Tighter spacing where available.")(func() {
		choice.Switch(c, &ic.CompactMode, choice.SwitchProps{})
	})
	rowSetting(c, tk, "Show icons", "Icons alongside labels.")(func() {
		choice.Switch(c, &ic.ShowIcons, choice.SwitchProps{})
	})
	rowSetting(c, tk, "Double-click action", "What a double-click on a request does.")(func() {
		ui.Box(c).Width(160).Shrink(0).Children(func() {
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
}

// pluginSettings lists available and enabled plugins.
func pluginSettings(c *ui.Context, tk *theme.Tokens, st *shellState) {
	pl := &st.settings.Plugins
	settingsGroup(c, tk, "Plugins", "Optional extensions bundled with Heartless.")
	if len(pl.Available) == 0 {
		ui.Text(c, "No plugins are available in this build.").FontSize(13).TextColor(tk.TextSecondary)
		return
	}
	for _, name := range pl.Available {
		name := name
		rowSetting(c, tk, name, "")(func() {
			enabled := contains(pl.Enabled, name)
			choice.Switch(c, &enabled, choice.SwitchProps{})
			pl.Enabled = setMembership(pl.Enabled, name, enabled)
		})
	}
}

// settingsGroup renders a section heading and caption.
func settingsGroup(c *ui.Context, tk *theme.Tokens, title, caption string) {
	ui.Column(c).AlignSelf(ui.Stretch).Gap(2).Children(func() {
		ui.Text(c, title).FontSize(15).FontWeight(600)
		if caption != "" {
			ui.Text(c, caption).FontSize(12).TextColor(tk.TextSecondary)
		}
	})
}

// rowSetting renders a labelled setting row with the control on the right.
func rowSetting(c *ui.Context, tk *theme.Tokens, label, hint string) func(control func()) {
	return func(control func()) {
		ui.Row(c).AlignSelf(ui.Stretch).MinWidth(0).MinHeight(40).Gap(16).AlignItems(ui.Center).
			PaddingY(4).Children(func() {
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, label).FontSize(13).TextColor(tk.Text)
				if hint != "" {
					ui.Text(c, hint).FontSize(11).TextColor(tk.TextTertiary).SingleLine()
				}
			})
			ui.Box(c).Shrink(0).Children(control)
		})
	}
}

// colorField renders a hex color input with a swatch preview.
func colorField(c *ui.Context, tk *theme.Tokens, value *string, placeholder string) {
	ui.Row(c).Width(220).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		swatch := tk.BorderSecondary
		if col, ok := parseHexColor(*value); ok {
			swatch = col
		}
		ui.Box(c).Width(20).Height(20).Radius(tk.BorderRadiusSM).Background(swatch).
			Border(1, tk.BorderSecondary).Shrink(0)
		ui.Box(c).Grow(1).Basis(0).MinWidth(0).Clip().Children(func() {
			field.Input(c, value, field.InputProps{Placeholder: placeholder, Size: theme.Small}).Grow(1)
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
	s.settingsDirty = false
	s.settingsSaved = true
	s.themeConfig = ThemeConfig(s.settings)
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
	out := list[:0]
	for _, item := range list {
		if item != v {
			out = append(out, item)
		}
	}
	return out
}

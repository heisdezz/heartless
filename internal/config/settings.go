package config

import "time"

// Theme holds theme configuration for the application interface.
type Theme struct {
	// Name of the theme (e.g., "light", "dark", "high-contrast").
	Name string `json:"name,omitempty"`

	// AccentColor is the primary accent color for UI elements.
	AccentColor string `json:"accent_color,omitempty"`

	// BackgroundColor is the main background color.
	BackgroundColor string `json:"background_color,omitempty"`

	// ForegroundColor is the main text color.
	ForegroundColor string `json:"foreground_color,omitempty"`

	// Inverted is whether the theme inverts light/dark.
	Inverted bool `json:"inverted,omitempty"`

	// CreatedAt tracks when the theme was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt tracks when the theme was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultTheme returns the default theme configuration.
func DefaultTheme() Theme {
	return Theme{
		Name:            "light",
		AccentColor:     "#0066ff",
		BackgroundColor: "#ffffff",
		ForegroundColor: "#000000",
		Inverted:        false,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
}

// Interface holds user interface configuration preferences.
type Interface struct {
	// ReduceMotion reduces animation motion for accessibility.
	ReduceMotion bool `json:"reduce_motion,omitempty"`

	// CompactMode uses compact UI layouts where available.
	CompactMode bool `json:"compact_mode,omitempty"`

	// ShowIcons controls whether icons are shown alongside text.
	ShowIcons bool `json:"show_icons,omitempty"`

	// DoubleClickAction action on double-click (e.g., "open", "edit", "none").
	DoubleClickAction string `json:"double_click_action,omitempty"`

	// FontSize sets the UI font size in DIPs; zero uses the theme's default.
	FontSize int `json:"font_size,omitempty" db:"font_size"`

	// CreatedAt tracks when the interface config was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt tracks when the interface config was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultInterface returns the default interface configuration.
func DefaultInterface() Interface {
	return Interface{
		ReduceMotion:      false,
		CompactMode:       false,
		ShowIcons:         true,
		DoubleClickAction: "open",
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// Plugins holds plugin configuration.
type Plugins struct {
	// Enabled lists plugins that are currently enabled.
	Enabled []string `json:"enabled,omitempty"`

	// Available lists plugins that are available but may be disabled.
	Available []string `json:"available,omitempty"`

	// Settings holds per-plugin configuration.
	Settings map[string]interface{} `json:"settings,omitempty"`

	// CreatedAt tracks when the plugin config was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt tracks when the plugin config was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultPlugins returns the default plugin configuration.
func DefaultPlugins() Plugins {
	return Plugins{
		Enabled:   []string{},
		Available: []string{},
		Settings:  map[string]interface{}{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// ApplicationSettings holds all application-level configuration.
type ApplicationSettings struct {
	// Theme is the visual theme configuration.
	Theme Theme `json:"theme,omitempty"`

	// Interface is the user interface preferences.
	Interface Interface `json:"interface,omitempty"`

	// Plugins holds plugin configuration.
	Plugins Plugins `json:"plugins,omitempty"`

	// CreatedAt tracks when the settings were created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt tracks when the settings were last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultApplicationSettings returns the default application settings.
func DefaultApplicationSettings() ApplicationSettings {
	return ApplicationSettings{
		Theme:     DefaultTheme(),
		Interface: DefaultInterface(),
		Plugins:   DefaultPlugins(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

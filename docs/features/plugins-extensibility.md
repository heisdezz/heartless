# Plugins and Extensibility

## Purpose

Extend client capabilities (auth, import/export, template functions, actions, themes) without hardcoding every integration into core.

## Observed Capabilities (Yaak reference)

- **Plugin platform**: `yaak-plugins` crate, plugin runtime/packages (`plugin-runtime`, `plugin-runtime-types`), plugin/key-value storage in models (`plugins`, `plugin_key_values`), install/source tracking migrations.
- **Extension points observed**:
  - **Authentication**: `auth-*` plugins.
  - **Importers**: `importer-*` plugins.
  - **Template functions**: `template-function-*` plugins.
  - **Actions/utilities**: `action-copy-curl`, `action-copy-grpcurl`, `action-send-folder`.
  - **Themes/UI assets**: `themes-yaak`.
- **External/community plugins**: `plugins-external/` directory and build/dev tooling in plugin packages.
- **Dynamic forms/config**: plugin-driven settings rendered via dynamic forms in UI.

## Key Components in Reference

- Crates/packages: `yaak-plugins`, `packages/plugin-runtime`, `packages/plugin-runtime-types`.
- Plugins: `yaak/plugins/*`, `plugins-external/`.
- Client: dynamic forms, action menus, template/auth integration points.

## Heartless Implementation Notes

- Per design doc, keep core functionality Go-native and GUI-independent; if extensibility is added, define stable interfaces and isolate runtimes/sandboxes.
- Prefer compiled/native extension points or well-bounded adapters over unrestricted scripting; enforce timeouts/capability limits and structured errors.
- Support discoverable plugin metadata (name, version, capabilities, config schema) and safe storage for plugin state.
- Do not require plugins for essential HTTP/auth/variable/history functionality; treat plugins as additive.

# UI and Productivity Features

## Purpose

Improve navigation, discoverability, configuration, and day-to-day workflows in the API client.

## Observed Capabilities (Yaak reference)

- **Command palette & shortcuts**: `CommandPaletteDialog.tsx`, `KeyboardShortcutsDialog.tsx`, hotkeys-related migrations/models.
- **Workspace/request navigation**: sidebars, recent requests/responses/connections dropdowns, switch/create/move dialogs, redirect to latest workspace.
- **Settings and preferences**: settings pages/dialogs, appearance/theme support (`theme.ts`, `themes-yaak`), font/config assets, proxy/CA/cert/DNS editors, sync/encryption settings.
- **Dialogs and workflows**: import/export, environment/request/folder settings, template function dialog, large-value/confirm dialogs, onboarding/help/error boundaries/toasts.
- **Response/request ergonomics**: URL bar, method selector, headers/params/body editors, markdown/Prose rendering, copy buttons, context menus, drag/drop markers.
- **Cross-platform client**: Tauri desktop app plus browser/web mode notes in development docs (IndexedDB-backed web host, limited network send).

## Key Components in Reference

- Client: extensive component library under `apps/yaak-client/components`, routes, hooks, command/init/theme modules.
- Packages: `@yaakapp-internal/ui`, `theme`, `platform` abstractions.

## Heartless Implementation Notes

- Build GUI as a consumer of application services/events (design constraint); keep transient UI state separate from domain persistence.
- Prioritize core workflows first: workspace tree, request editor, execution controls, response/history/timeline, environment/auth editors.
- Use Lobe MyGO components and MyGO patterns for dialogs, command surfaces, theming, and layout; maintain accessibility and keyboard navigation.
- Gate advanced/optional features (Git, plugin marketplace/runtime, browser parity) behind clear service interfaces and incremental milestones.

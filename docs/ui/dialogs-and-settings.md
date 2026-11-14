# Dialogs, Command Palette, and Settings

Global/supporting UI surfaces for navigation, configuration, data transfer, and focused tasks.

## Command Palette

- **Purpose**: fast keyboard-first access to actions (open request, switch workspace/environment, send, create entity, toggle panels, run commands).
- **Layout**: modal overlay with search input, filtered action/result list, shortcut hints, and context-aware ranking.
- **Behavior**: non-blocking to background state except focus; executes commands through application/UI command handlers.

## Settings

- **Scope**: application settings and workspace/project settings (separate where behavior differs).
- **Areas** (incremental): appearance/theme, editor behavior, network/proxy/TLS/DNS defaults, request defaults/timeouts, history retention, data/storage paths, sync/encryption, plugins/extensibility when available.
- **Layout**: navigable sections/categories, immediate or explicit-save controls with clear feedback, sensitive fields masked/redacted.

## Import / Export

- **Import dialog**: choose source/format/file or paste (e.g. cURL), preview/mapping summary, conflict/unsupported-item report, target workspace, transactional apply.
- **Export dialog**: select scope/entities and format, secret-handling options/policy warnings, destination.
- **Progress**: long-running imports/exports show status and allow cancellation where safe.

## Core Dialogs

- **Create/edit**: workspace, folder, request, environment dialogs with validation and minimal required fields.
- **Move/duplicate/delete confirmations**: show impact (children/history references as relevant) and require explicit confirmation for destructive actions.
- **Git/sync dialogs** (when enabled): clone, commit/push/pull status, conflict resolution entry points, filesystem sync configuration.
- **Utility dialogs**: cookie manager, certificate/DNS/proxy editors, large value/body preview, template/function helpers, keyboard shortcuts, about/diagnostics.

## Notifications and Feedback

- **Toasts/banners**: success, warning, recoverable error, background task completion; actionable when relevant (open details/retry).
- **Inline errors**: field-level validation in editors/dialogs; structured diagnostics in response/timeline/import reports.
- **Progress**: determinate where possible (import/sync), indeterminate for unknown-duration network operations, always with cancel for user-initiated tasks.

## Yaak Reference Mapping

- `CommandPaletteDialog.tsx`, `KeyboardShortcutsDialog.tsx`, `Settings*`, `WorkspaceSettingsDialog.tsx`, `ImportDataDialog.tsx`, `ImportCurl.tsx`, `ExportDataDialog.tsx`, `Create*Dialog.tsx`, `MoveToWorkspaceDialog.tsx`, `CookieDialog.tsx`, `DnsOverridesEditor.tsx`, `CloneGitRepositoryDialog.tsx`, `Toasts.tsx`, `ErrorBoundary.tsx`.
- Heartless should prioritize dialogs needed for Phase 0–2 workflows (workspace/request/environment basics, HTTP send settings, history) before advanced sync/Git/plugin surfaces.

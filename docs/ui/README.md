# UI Layout Documentation

Layout and screen-structure documentation for the Heartless desktop client (MyGO + Lobe MyGO). These docs describe UI regions, responsibilities, and state boundaries; they do not prescribe pixel-perfect visuals.

## Principles

- **GUI consumes application services/events**: domain execution, storage, and protocol logic live outside the UI layer.
- **Separate persistent vs transient state**: saved workspace/request data is persisted; selection, drafts, in-flight executions, connection state, and panel layout are transient UI/app state.
- **Protocol-specific panes, shared chrome**: HTTP/GraphQL/gRPC/WebSocket have specialized editors/viewers but share workspace navigation, history, environment/auth context, and settings.
- **Cancelable long-running work**: sends, streams, imports, sync, and Git actions must surface progress and cancellation.
- **One component per file**: don't pile all UI components into a single file. Each component, screen, dialog, or reusable widget lives in its own file, grouped by feature/region (e.g. `screens/`, `components/sidebar/`, `components/dialogs/`). A file should contain one primary component plus only its small private helpers; extract shared pieces into their own files.

## Documents

| Document                                           | Scope                                                                |
| -------------------------------------------------- | -------------------------------------------------------------------- |
| [app-shell.md](app-shell.md)                       | Window, title bar, top-level regions, global overlays                |
| [workspace-sidebar.md](workspace-sidebar.md)       | Workspace header, collection/folder/request tree, navigation actions |
| [request-editor.md](request-editor.md)             | Request authoring layouts for HTTP/GraphQL/gRPC/WebSocket            |
| [response-panel.md](response-panel.md)             | Response viewers, metadata/timeline, history and recent results      |
| [environments-auth.md](environments-auth.md)       | Environment/variable management and authentication configuration     |
| [dialogs-and-settings.md](dialogs-and-settings.md) | Command palette, settings, import/export, and supporting dialogs     |

## Related Docs

- [../PLAN.md](../PLAN.md) — implementation roadmap and architectural guardrails
- [../features/README.md](../features/README.md) — feature capabilities derived from Yaak reference

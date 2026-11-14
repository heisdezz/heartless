# Workspaces, Collections, and Organization

## Purpose

Provide hierarchical organization for API work: separate projects/workspaces, group requests into folders/collections, and manage saved request definitions independently from execution state.

## Observed Capabilities (Yaak reference)

- **Workspaces**: top-level containers for requests, environments, settings, and metadata (`yaak-models` workspace/workspace_meta models and migrations; `Workspace*` UI components/dialogs/actions).
- **Folders**: nested grouping of requests (`Folder` model, folder layout/settings dialogs, sidebar tree).
- **Request entities**: distinct typed request models (HTTP, GraphQL handled over HTTP, gRPC, WebSocket) with IDs, names, descriptions, sort order, folder/workspace association, and timestamps (see `yaak-models/src/models.rs`, query modules).
- **Ordering and navigation**: sort priority, sidebar organization, recent requests/workspaces, create/move/duplicate/switch flows (`Sidebar*`, `RecentRequestsDropdown`, `MoveToWorkspaceDialog`, `SwitchWorkspaceDialog`, `CreateWorkspaceDialog`).
- **Saved vs transient state**: request definitions persisted separately from connection/response execution artifacts (connections/events/history models and tables).

## Key Components in Reference

- Models/queries: `yaak-models` (`workspaces`, `workspace_metas`, `folders`, `http_requests`, `grpc_requests`, `websocket_requests`, `batch`, `duplicate_name`).
- Client UI: `Workspace.tsx`, `WorkspaceHeader.tsx`, `WorkspaceSettingsDialog.tsx`, `Sidebar.tsx`, `FolderLayout.tsx`, `FolderSettingsDialog.tsx`, `MoveToWorkspaceDialog.tsx`.

## Heartless Implementation Notes

- Model workspace/collection/folder/request as domain entities with clear ownership and inheritance points (auth, variables, settings).
- Keep GUI state (selection, expanded nodes, drafts) separate from persisted domain state.
- Support transactional operations for moves/duplicates/imports; enforce ordering/indexing in SQLite.
- Align with design doc precedence/inheritance rules and avoid storing response data on saved requests.

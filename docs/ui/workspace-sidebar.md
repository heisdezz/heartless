# Workspace Sidebar Layout

Navigation and organization surface for workspaces, collections/folders, and saved requests.

## Regions

```text
+----------------------------------+
| Workspace header                 |
| [icon/name] [switch] [actions]   |
+----------------------------------+
| Quick filters / search (optional)|
+----------------------------------+
| Collection tree (scroll)         |
|  v Workspace / Collection        |
|    v Folder                      |
|      * HTTP Request              |
|      * GraphQL Request           |
|      * gRPC Request              |
|      * WebSocket Request         |
|    > Folder (collapsed)          |
+----------------------------------+
| Footer actions (settings/sync)   |
+----------------------------------+
```

## Responsibilities

- **Workspace header**: current workspace identity, switch/create/manage actions, environment selector shortcut if appropriate.
- **Tree**: hierarchical folders and typed requests with icons/badges (method/protocol, unsaved/dirty, active). Supports expand/collapse, reorder, and selection.
- **Item actions**: context menu / inline actions for create, rename, duplicate, move, delete, copy/export (where supported).
- **Footer/utility area**: workspace settings, environments, sync/Git status or entry points when enabled.

## Interaction Behavior

- Single active selection drives primary content; multi-select can be deferred until bulk operations are required.
- Drag-and-drop for reorder/move with clear drop markers and transactional persistence.
- Keyboard navigation: up/down, expand/collapse, open, rename, delete (with confirmation).
- Collapsed/rail mode shows compact affordances (icons/avatars) and preserves access to active item; expanded mode shows labels/metadata.
- Search/filter narrows visible tree without mutating persisted structure.

## Data and State

- **Persistent (domain)**: workspace, folders, requests, ordering, metadata.
- **Transient (UI)**: expanded nodes, scroll position, selection highlight, rename-in-progress, drag state, filter text.
- **Derived**: request display labels, method/protocol badges, inherited indicators (auth/env) where useful.

## Yaak Reference Mapping

- `Workspace.tsx`, `WorkspaceHeader.tsx`, `Sidebar.tsx`, `SidebarActions.tsx`, `FolderLayout.tsx`, recent/switch/move/create dialogs.
- Heartless should keep tree rendering in GUI but CRUD/move/duplicate operations go through application services/repositories.

# App Shell Layout

Top-level window structure hosting workspace navigation, active request/response work area, and global overlays.

## Regions

```text
+------------------------------------------------------------------+
| Title Bar (window drag, app/workspace identity, window controls) |
+------------------------------------------------------------------+
| [Sidebar / Nav] | [Primary Workspace Content]                    |
|                 |  +-------------------------------------------+ |
|                 |  | Context header (tabs/breadcrumb/actions)  | |
|                 |  +----------------------+--------------------+ |
|                 |  | Request Editor       | Response / Stream  | |
|                 |  | (protocol-specific)  | Panel              | |
|                 |  +----------------------+--------------------+ |
+------------------------------------------------------------------+
| Optional status/toast region (transient)                         |
+------------------------------------------------------------------+
```

## Responsibilities

- **Title bar**: app name/workspace context, draggable area, platform window controls/insets. Keep minimal; avoid duplicating primary navigation.
- **Sidebar**: collapsible/resizable navigation (`DraggableSideNav`/equivalent). Owns workspace tree visibility and quick actions; see [workspace-sidebar.md](workspace-sidebar.md).
- **Primary content**: renders selected entity (HTTP/gRPC/WebSocket request, folder/workspace settings view). Layout adapts by protocol and available width.
- **Global overlays**: command palette, dialogs, toasts/notifications, transient progress indicators. Overlays must not own domain state.

## Layout Behavior

- Sidebar width/expanded state persisted as UI preference (per workspace/window as appropriate), not as domain data.
- Request/response split should be resizable (horizontal or vertical depending on viewport/protocol) with sensible defaults and collapse behavior.
- Empty states: no workspace, empty collection, no request selected, no response yet—each with a clear primary action.
- Keyboard focus and shortcuts operate at shell level (palette, navigation, send/cancel) and delegate to active pane.

## State Boundaries

- **Persistent**: active workspace/request identifiers (as needed for restore), UI layout preferences.
- **Transient**: open dialogs, command palette query, in-flight execution indicators, websocket/gRPC connection status, toast queue.
- **Events**: shell subscribes to typed app events for execution/connection/sync/errors and updates transient indicators only.

## MyGO / Lobe MyGO Notes

- Use `ui.Row`/`ui.Column` with fill/grow/basis for main regions; `DraggableSideNav` for sidebar resize/collapse.
- Apply theme tokens via `theme.Apply`/`theme.TokensOf`; avoid hardcoded colors outside tokenized component props.
- Marshal background/service updates onto UI-safe update paths per MyGO requirements.

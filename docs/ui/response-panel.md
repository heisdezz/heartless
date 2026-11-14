# Response and History Panel

Inspection surface for execution results, streaming events, protocol messages, and prior runs.

## Common Structure

```text
+------------------------------------------------------------------+
| Response header: status/state, time, size, actions (copy/save)   |
+------------------------------------------------------------------+
| Tabs: Body | Headers/Metadata | Timeline/Events | Cookies | ...  |
+------------------------------------------------------------------+
| Viewer/content area                                               |
| (pretty/raw/preview, tables, event log, stream messages)         |
+------------------------------------------------------------------+
```

## HTTP / GraphQL Responses

- **Summary**: status code/text, duration/timing breakdown, response size, effective URL/redirect info.
- **Body viewers**: pretty/structured (JSON/XML/HTML), raw, hex/preview for binary, media/PDF/markdown previews where supported.
- **Headers/metadata**: response headers, request headers as sent (redacted as needed), cookies set/received.
- **Timeline/events**: ordered execution events (settings applied, DNS/TLS/connect, request/response phases, errors) for diagnostics.
- **Filtering/search**: JSONPath/XPath or text search/filter within body for inspection (read-only over response data).

## gRPC Responses

- **Call summary**: status/code, timing, method/service metadata.
- **Messages/events**: unary response or streamed message list with timestamps and direction; trailers/metadata.
- **Schema-aware rendering**: decoded/pretty message view based on reflected/provided descriptors.

## WebSocket / Streaming

- **Connection summary**: URL, status (connecting/open/closed/error), durations/reconnect info.
- **Message/event log**: chronological sent/received frames/events with filtering, clear/pause controls, payload preview.
- **Large streams**: bounded in-memory rendering, virtualized lists, optional persistence policy.

## History and Recent Results

- **Recent results**: quick access to previous responses/connections for the active request or workspace context.
- **History records**: persisted execution metadata and optionally bodies/events per retention policy; open/read without mutating saved request.
- **Comparison (future-capable)**: layout should allow side-by-side or diff affordances later, but initial implementation can focus on single-result inspection.

## Behavior and State

- **Transient vs persisted**: live streaming buffers may be transient; completed results/history persisted according to storage/retention rules.
- **Large payloads**: warn/confirm on very large bodies, stream to disk/blob storage, avoid freezing UI.
- **Errors**: distinguish transport/protocol failures from valid HTTP/gRPC error responses; show structured error details and redacted diagnostics.
- **Cancellation**: visible cancel/stop controls must propagate to execution/connection contexts.

## Yaak Reference Mapping

- `HttpResponsePane.tsx`, `HttpResponseTimeline.tsx`, `ResponseInfo.tsx`, `responseViewers/*`, `GrpcResponsePane.tsx`, `WebsocketResponsePane.tsx`, `RecentHttpResponsesDropdown.tsx`, `RecentGrpcConnectionsDropdown.tsx`, `RecentWebsocketConnectionsDropdown.tsx`.

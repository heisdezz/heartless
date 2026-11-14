# WebSocket and Server-Sent Events (SSE)

## Purpose

Support persistent/streaming API interactions: bidirectional WebSocket sessions and server-push event streams (SSE).

## Observed Capabilities (Yaak reference)

### WebSocket
- **Dedicated crate**: `yaak-ws` (`connect`, `manager`, `render`, request rendering) built on `tokio-tungstenite` with TLS/template/model integration.
- **Request/connection/event persistence**: websocket request, connection, and event models/queries (`websocket_requests`, `websocket_connections`, `websocket_events`).
- **UI workflows**: request/response panes, connection layout, recent connections (`WebsocketRequestLayout`, `WebsocketRequestPane`, `WebsocketResponsePane`, `RecentWebsocketConnectionsDropdown`).
- **Session behavior**: connect/disconnect lifecycle, send/receive messages, headers/auth/URL rendering, cancellable operations implied by async manager design.

### SSE
- **Dedicated crate**: `yaak-sse` providing typed SSE-related structures/serialization (lightweight crate; used alongside HTTP streaming/event handling).
- **Event-oriented responses**: fits response-event/timeline capture patterns used for streamed HTTP interactions.

## Key Components in Reference

- Crates: `yaak-ws`, `yaak-sse`, `yaak-http`/`yaak-tls` dependencies for transport concerns.
- Client: WebSocket layout/panes and recent connection tracking.
- Models: websocket and HTTP response event entities/migrations.

## Heartless Implementation Notes

- Implement as protocol adapters with explicit connection state machines and typed app events (`connected`, message received/sent, `disconnected`, errors).
- Ensure all connections/streams honor `context.Context` cancellation and avoid goroutine leaks.
- Stream/bound message buffers for large or high-frequency traffic; persist history/events according to retention policy.
- SSE can be modeled as HTTP response streaming with parsed event frames feeding the same timeline/inspection UX.

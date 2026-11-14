# Response Inspection and History

## Purpose

Capture, persist, and present request/connection outcomes (status, metadata, headers, bodies, events, timing) and allow users to review recent executions.

## Observed Capabilities (Yaak reference)

- **Structured response/event storage**: HTTP responses, HTTP response events/timeline, gRPC connections/events, WebSocket connections/events, response headers/content-length/message size/DNS timing fields (model/query modules and migrations).
- **Execution timeline**: timeline view for HTTP responses showing send/settings/events progression (`HttpResponseTimeline.tsx`, `ResolvedHttpRequestSettings.timeline_events`).
- **Response viewers**: multiple viewers/renderers for bodies and formats (`components/responseViewers/*`, syntax highlighting/PDF/markdown deps, binary handling support).
- **Metadata display**: status, timing, size, headers, effective request/response info (`ResponseInfo.tsx`, panes).
- **History/recent access**: recent HTTP responses, gRPC connections, WebSocket connections, recent requests dropdowns; history retention/management implied by persisted entities and settings.
- **Large payload handling**: blob/body storage considerations (`blob_manager`, blob migrations), request/response size settings and confirmation UX (`ConfirmLargeRequestBody.tsx`).

## Key Components in Reference

- Crates/models: `yaak-models` (`http_responses`, `http_response_events`, `grpc_connections/events`, `websocket_connections/events`, blob manager).
- Client: `HttpResponsePane.tsx`, `GrpcResponsePane.tsx`, `WebsocketResponsePane.tsx`, `ResponseInfo.tsx`, `Recent*Dropdown.tsx`, `responseViewers/`.

## Heartless Implementation Notes

- Return structured `ExecutionResult` objects; persist history separately from saved request definitions.
- Support text and binary bodies, streaming where possible, and offloading large payloads from main DB rows when appropriate.
- Make retention configurable (metadata vs bodies) and provide clear history operations.
- Distinguish transport failures from HTTP error responses in UI and error taxonomy.

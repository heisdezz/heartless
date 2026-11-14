# Request Editor Layout

Protocol-specific authoring surface for creating, editing, and sending API requests.

## Common Structure

```text
+------------------------------------------------------------------+
| Request context bar: name, method/type, URL/target, Send/Cancel  |
+------------------------------------------------------------------+
| Tabs/sections: Params | Headers | Auth | Body | Settings | ...  |
+------------------------------------------------------------------+
| Active editor/content area                                        |
| (key-value tables, code editors, schema pickers, message log)    |
+------------------------------------------------------------------+
```

## HTTP / GraphQL

- **URL bar**: method selector, URL/editor with variable highlighting, send/cancel actions.
- **Sections**: query params, headers, authentication, body (none/form/JSON/text/binary/graphql as applicable), pre/post or script hooks (when enabled), request settings (timeout, redirects, TLS validation, proxy, cookies, HTTP version).
- **GraphQL specialization**: query/mutation editor, variables editor, schema/introspection controls, operation selection; still sent via HTTP transport settings.
- **Editors**: template-aware inputs, environment variable resolution preview (non-secret-safe), validation feedback.

## gRPC

- **Target/config**: server address, TLS/metadata/auth, reflection vs proto/schema source selection.
- **Method selection**: service/method browser, streaming mode indicators (unary/client/server/bidi).
- **Message editor**: structured/JSON message input with schema hints; metadata/headers editor.
- **Execution controls**: invoke/send/cancel/end-stream as appropriate for streaming modes.

## WebSocket

- **Connection config**: URL, headers/auth/params, connect/disconnect controls.
- **Message composer**: send text/binary payloads, saved/reusable messages (optional).
- **Session view**: outbound/inbound message log integrated with or adjacent to response/stream panel.

## Behavior and State

- **Saved draft vs persisted request**: edits bind to a draft model; explicit save or autosave policy must be consistent and visible (dirty indicator).
- **Validation**: inline errors for URL, JSON/body, missing schema, invalid settings; block send only when execution cannot proceed safely.
- **Variable/auth inheritance**: show effective source (request/folder/collection/environment/workspace) where helpful, without exposing secret values.
- **Transient execution state**: sending, connected/connecting/disconnected, per-request progress/cancel handle.

## Yaak Reference Mapping

- `HttpRequestLayout.tsx`, `HttpRequestPane.tsx`, `UrlBar.tsx`, `HeadersEditor.tsx`, `UrlParameterEditor.tsx`, `HttpAuthenticationEditor.tsx`, `GraphqlEditor` components, `GrpcRequestLayout/Pane`, `GrpcProtoSelectionDialog`, `WebsocketRequestLayout/Pane`.
- Heartless must route send/connect actions to application services with `context.Context` cancellation and render results from structured events/results.

# HTTP Client and Execution

## Purpose

Author, send, and inspect HTTP requests with configurable transport behavior, cancellation, and structured result capture.

## Observed Capabilities (Yaak reference)

- **Request authoring**: method, URL, query parameters, headers, body/content types, descriptions, naming/organization (`HttpRequest*` model fields, `HttpRequestPane`, `HeadersEditor`, `UrlBar`, `UrlParameterEditor`, request editors).
- **Transport/execution**: native HTTP execution (`yaak-http` crate; deps indicate `reqwest` with rustls/native-tls, HTTP/2, SOCKS/proxy support, compression/streaming helpers, cookie handling integration points).
- **Configurable send settings**: timeout, redirects/follow behavior, certificate validation, cookie send/store, HTTP version selection, request/message size limits (`ResolvedHttpRequestSettings`, `HttpSendSettings`, `http_version`/timeline settings in models and migrations).
- **Response capture**: status/headers/body/metadata and timing-related data (`http_responses`, `http_response_events`, response request headers/content-length/DNS timing migrations), timeline/response panes.
- **Cancellation and lifecycle**: request execution managed outside UI rendering concerns; recent responses/connections tracked.

## Key Components in Reference

- Crates: `yaak-http`, `yaak-tls`, `yaak-models` (HTTP request/response/event models and queries).
- Client UI: `HttpRequestLayout.tsx`, `HttpRequestPane.tsx`, `HttpResponsePane.tsx`, `HttpResponseTimeline.tsx`, `ResponseInfo.tsx`, `RecentHttpResponsesDropdown.tsx`, `ConfirmLargeRequestBody.tsx`.

## Heartless Implementation Notes

- Implement the design-doc execution pipeline as a GUI-independent service using `net/http` transport adapters and `context.Context`.
- Resolve variables/auth before building the outbound request; persist immutable execution results/history records.
- Stream large request/response bodies and avoid per-request transport re-creation where pooling applies.
- Record structured network/validation errors distinctly from HTTP 4xx/5xx responses.

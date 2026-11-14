# Yaak Feature Reference

Documentation of Yaak features derived from the `yaak/` reference tree. This is input for Heartless (Go/MyGO) implementation planning; see [`../PLAN.md`](../PLAN.md) for the phased roadmap.

## Feature Index

| Feature | Doc | Primary areas in Yaak |
|---|---|---|
| Workspaces, folders, requests, organization | [workspaces-collections.md](workspaces-collections.md) | `yaak-models`, client workspace/sidebar components |
| HTTP client & execution | [http-client.md](http-client.md) | `yaak-http`, `yaak-tls`, client HTTP panes/editors |
| GraphQL | [graphql.md](graphql.md) | client GraphQL components/editor, HTTP transport |
| gRPC | [grpc.md](grpc.md) | `yaak-grpc`, client gRPC panes/dialogs |
| WebSocket & SSE | [websocket-sse.md](websocket-sse.md) | `yaak-ws`, `yaak-sse`, client WebSocket panes |
| Environments & variables | [environments-variables.md](environments-variables.md) | `yaak-models`, `yaak-templates`, environment editors |
| Authentication | [authentication.md](authentication.md) | `plugins/auth-*`, `HttpAuthenticationEditor` |
| Templating & dynamic values | [templating.md](templating.md) | `yaak-templates`, `plugins/template-function-*` |
| Response inspection & history | [response-inspection-history.md](response-inspection-history.md) | response models/events, viewers, timeline, recent/history UI |
| Filtering & querying response data | [filtering-search.md](filtering-search.md) | `yaak-filter` (JSONPath/XPath), response viewers |
| Import & export | [import-export.md](import-export.md) | `plugins/importer-*`, import/export dialogs, sync models |
| Filesystem sync & Git | [sync-git.md](sync-git.md) | `yaak-sync`, `yaak-git`, sync/Git UI components |
| Cookies & session data | [cookies-storage.md](cookies-storage.md) | cookie models/dialog/dropdown, HTTP send/store settings |
| Network, TLS, proxy, DNS | [network-security-tls.md](network-security-tls.md) | `yaak-tls`, `yaak-proxy`, request/workspace network settings |
| Secrets & encryption | [secrets-encryption.md](secrets-encryption.md) | `yaak-crypto`, keychain/secure storage, workspace encryption UI |
| Plugins & extensibility | [plugins-extensibility.md](plugins-extensibility.md) | `yaak-plugins`, plugin runtime/types, action/auth/import/template/theme plugins |
| UI & productivity | [ui-productivity.md](ui-productivity.md) | command palette, themes, hotkeys, dialogs, settings, onboarding |

## Notes

- Yaak reference stack: Tauri + Rust backend crates + React/TypeScript client + Node/plugin runtime.
- Heartless target stack differs (Go-native core, MyGO/Lobe MyGO GUI); docs capture behavior/capabilities to preserve, not implementation details to copy verbatim.
- Where Yaak relies on a JS plugin/runtime model, Heartless design favors a native-first core and optional isolated scripting/runtime adapters.

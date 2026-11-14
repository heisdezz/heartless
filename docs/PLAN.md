# Heartless Implementation Plan

Plan derived from the Yaak-inspired API client design document, adapted to the current `heartless` scaffold.

## Current State

- MyGO desktop shell in `main.go` with a sample window/view.
- Initial dashboard screen in `internal/screens/dash.go` using `lobe-mygo` `DraggableSideNav`.
- Empty domain/state scaffold in `internal/state/app_state.go`.
- Dependencies: `github.com/egoist/mygo v0.9.0`, `github.com/lobehub/lobe-mygo`.
- No domain, service, transport, storage, or migration packages yet.

## Architectural Guardrails

- Go-native core; MyGO/Lobe MyGO are GUI consumers only.
- Domain logic must not depend on GUI, SQLite, or concrete transports.
- Services depend on interfaces (repositories, transports, auth providers, script engines).
- Centralized variable resolution and modular authentication providers.
- Request execution independent of GUI state; all long-running work accepts `context.Context`.
- Structured errors, `log/slog` with redaction, secure secret handling.
- Streaming/temporary storage for large bodies; bounded concurrency.

## Target Package Layout (introduce incrementally)

```text
cmd/heartless/            # optional later; main.go remains bootstrap for now
internal/app/             # bootstrap, lifecycle, wiring
internal/domain/          # request, response, collection, environment, workspace,
                          # auth, history, assertion, variable
internal/services/        # request, collection, environment, history, workspace,
                          # importer, exporter
internal/transport/       # http, websocket, graphql, grpc adapters
internal/auth/           # basic, bearer, apikey, oauth2, digest providers
internal/scripting/       # engine interface, runtime adapters, execution sandbox
internal/storage/         # sqlite repositories, filesystem, secrets
internal/importer/        # postman, insomnia, openapi
internal/exporter/
internal/events/          # typed app events
internal/config/
internal/platform/
gui/ or internal/gui/     # application shell, components, pages, state
migrations/
tests/integration/
tests/fixtures/
```

## Phased Roadmap

### Phase 0 — Foundation Alignment (immediate)

- [ ] Rename/wire app identity consistently (`heartless`) and stabilize window/bootstrap.
- [ ] Introduce `internal/app` bootstrap/lifecycle and a GUI/application service boundary.
- [ ] Define core domain models and IDs (workspace, collection/folder, request, environment/variable, response/execution result, history entry).
- [ ] Add typed event bus/contracts and transient execution state model.
- [ ] Establish config/logging (`log/slog`) with sensitive-value redaction rules.
- [ ] Set up SQLite storage layer skeleton, repository interfaces, migrations framework, FK enforcement, transactions.
- [ ] Replace `internal/state/app_state.go` placeholder with purposeful app/GUI state separation.

**Exit criteria:** app boots, domain/service/storage interfaces compile, migrations run, GUI reads through services/events only.

### Phase 1 — HTTP Core

- [ ] HTTP request/response domain models (method, URL, params, headers, body, timeout, redirects, TLS/proxy config).
- [ ] Request validation and deterministic variable interpolation/resolution pipeline with scope precedence and cycle/missing-value policy.
- [ ] `net/http`-based transport adapter (HTTP/1.1, HTTP/2, TLS, redirects, proxy, pooling, cancellation, timeouts, streaming).
- [ ] Execution pipeline: validate -> resolve -> auth apply -> pre-process -> build -> transport -> capture -> assert/post-process -> persist history -> return result.
- [ ] Structured execution result and error taxonomy (validation, resolution, auth, DNS/connect/TLS/timeout/HTTP exec, response processing, storage).
- [ ] Execution history persistence with retention hooks and large-body handling strategy.

**Exit criteria:** send/cancel HTTP requests from GUI, resolved copies only, history recorded, tests via `httptest`.

### Phase 2 — Collections and Environments

- [ ] Workspace/collection/folder CRUD with ordering, metadata, nesting.
- [ ] Saved requests decoupled from transient response/execution state.
- [ ] Environment management, variable scopes/types/secret flags, inheritance/override rules.
- [ ] Collection/folder/request-level auth and variable inheritance.
- [ ] Repository queries/indexes and transactional multi-entity operations.

### Phase 3 — Authentication

- [ ] Provider interface and request application pipeline.
- [ ] None, Basic, Bearer, API key, Digest providers.
- [ ] OAuth 2.0 flows/token refresh service (isolated) and secure credential storage abstraction (OS keychain + encrypted fallback policy).
- [ ] Secret separation from ordinary metadata; redaction in logs/history/export previews.

### Phase 4 — API Testing and Extensibility

- [ ] Native assertions (status, headers, body, JSON fields, duration) independent of scripting.
- [ ] Pre-request/post-response processing hooks and controlled execution context.
- [ ] Optional scripting runtime interface; Goja adapter only if needed, sandboxed, time-limited, no unrestricted FS/network/process access.
- [ ] Structured script/runtime errors surfaced without destabilizing app.

### Phase 5 — Additional Protocols

- [ ] GraphQL over HTTP (queries, mutations, variables, headers, error inspection).
- [ ] WebSocket adapter (lifecycle, send/receive, headers/auth, reconnect controls, connection state events).
- [ ] gRPC adapter incrementally (service discovery, unary, streaming) without leaking into generic request model.

### Phase 6 — Interoperability

- [ ] Import pipeline: parse -> validate -> map -> report unsupported/conflicts -> normalized result -> transactional persist.
- [ ] Importers: Postman, Insomnia, OpenAPI. Exporters: native workspace format (+ selected targets as needed).
- [ ] Secret/export policy (exclude or explicit secure handling), relationship/structure preservation.

### Phase 7 — GUI Maturity

- [ ] Workspace/collection tree, request editor, response/history views, environment/auth editors driven by application services and events.
- [ ] Transient execution/connection/loading/error state in GUI layer only; marshal UI updates per MyGO requirements.
- [ ] Cancellation/progress UX for long-running operations.

### Phase 8 — Optimization, Security, and Validation

- [ ] Streaming/large payload handling, retention tuning, DB/index/query optimization.
- [ ] Concurrency hardening (bounded workers, no leaked goroutines), profiling with realistic workloads.
- [ ] TLS defaults/overrides, secure local data permissions, sensitive-data export/backup policy.
- [ ] Cross-platform validation.

## Testing and Quality Gates

- Unit tests for domain, resolver, auth providers, serialization, services.
- Transport tests with `httptest` (construction, headers, auth, redirects, timeouts, parsing, cancellation, streaming).
- Integration tests: services + repositories + SQLite + transports + resolution + history.
- Import/export fixture tests and round-trip checks for supported fields.
- Concurrency/race coverage for execution, persistence, cancellation.
- Baseline commands: `go test ./...`, `go test -race ./...`, `go vet ./...` (plus lint/typecheck commands if added to repo).

## Definition of Done (per feature)

- Domain behavior and service implementation defined, GUI-independent.
- Consistent structured errors and `context.Context` cancellation where applicable.
- Sensitive data handling/redaction verified.
- Unit/integration tests for important interactions.
- Documentation updated; no unnecessary coupling introduced.

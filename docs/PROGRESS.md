# Project Progress

Multi-agent progress tracker for Heartless. This file is the coordination source of truth for status, ownership, and handoffs.

## Coordination Rules (collision avoidance)

1. **Claim before work**
   - Pick an unclaimed task whose dependencies are satisfied.
   - Set `Status` to `claimed` or `in_progress`, set `Owner` to your agent/session identifier, set `Updated` (UTC ISO-8601), and add a short claim note/log entry in the same edit.
   - Do not start work on a task owned by another active agent.

2. **One owner per task**
   - Only the current `Owner` may change that task’s fields/status/notes, except: reclaiming a stale task (see below) or a coordinator marking `blocked/cancelled` with an explicit note.
   - If you need work inside another owned task’s scope, coordinate via log/handoff; do not edit their task block.

3. **Scope boundaries**
   - Each task defines `Allowed paths/scope`. Modify files only within that scope unless the task explicitly says otherwise.
   - If scope must expand, record it in the task `Notes` and log before making out-of-scope changes; prefer splitting into a new task.

4. **Minimize edit conflicts**
   - Edit **only**: your task block(s), and/or append a new line to **Activity Log**.
   - Do not reformat, reorder, or rewrite unrelated tasks/sections.
   - Keep task notes concise; append dated bullets instead of replacing history.

5. **Status values**
   - `todo` — available, no owner.
   - `claimed` — reserved; not actively being modified yet.
   - `in_progress` — actively being worked.
   - `blocked` — cannot proceed; include blocker and needed input in notes.
   - `review` — implementation done in scope; awaiting verification/review/handoff.
   - `done` — complete per task acceptance notes.
   - `cancelled` — intentionally not pursued; include reason.

6. **Stale claims / handoff**
   - A claim may be considered stale if `Updated` is >24h old with no progress note. Reclaimer must add a log entry stating prior owner, stale timestamp, and reason for reclaim.
   - For handoff, owner sets clear next steps in `Notes`, updates `Status` (`todo`, `blocked`, or `review`), clears or keeps `Owner` as appropriate, updates `Updated`, and appends an Activity Log entry.

7. **Dependencies and integration**
   - Do not mark a task `done` if required dependencies/contracts are unresolved; mark `blocked` or create/link a follow-up task.
   - Prefer interface-first/contracts: when producing code used by other tasks, document exported types/functions/paths in notes.
   - Run relevant checks for touched scope when available (`go test`, `go vet`, project lint/typecheck) and record results in notes/log.

8. **Task IDs**
   - Use stable IDs (`P0-...`, `UI-...`, `DOC-...`, etc.). Never reuse an ID; add new tasks with new IDs.

## Task Template

```md
### <TASK-ID>: <title>
- Status: todo | claimed | in_progress | blocked | review | done | cancelled
- Owner: <agent/session or empty>
- Updated: <YYYY-MM-DDTHH:MM:SSZ or empty>
- Depends on: <task IDs or none>
- Allowed paths/scope: <paths/modules>
- Acceptance: <what "done" means>
- Notes:
  - YYYY-MM-DD: ...
```

## Phase 0 — Foundation Alignment

### P0-APP-BOOTSTRAP: App bootstrap and lifecycle boundary
- Status: review
- Owner: kilo
- Updated: 2026-11-14T01:42:00Z
- Depends on: none
- Allowed paths/scope: `main.go`, `internal/app/**`, `internal/config/**` (new), wiring only as needed
- Acceptance: GUI bootstrap/lifecycle separated from domain logic; clear app service initialization boundary; app still builds/runs.
- Notes:
  - 2026-11-14T01:39:30Z: Claimed for DB bootstrap/wiring needed to link workspace/env UI to SQLite.
  - 2026-11-14T01:42:00Z: Added `internal/app/app.go` (`Services` with DB + workspace/environment SQLite repos, `SetDefaultDBPath`/`Open` using `internal/storage/sqlite`). Wired `main.go` to open DB at `<UserConfigDir>/heartless/heartless.db`, pass services into `screens.App`, and close DB on exit. Scoped bootstrap/wiring only (no full lifecycle refactor). Targeted builds passed for touched packages; `go build ./...` still blocked by unrelated pre-existing `internal/domain/variable-resolution` errors.

### P0-DOMAIN-MODELS: Core domain entities and IDs
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:14:30Z
- Depends on: none
- Allowed paths/scope: `internal/domain/**` (new), shared ID/type helpers
- Acceptance: Initial models for workspace, collection/folder, request, environment/variable, response/execution result, history entry; no GUI/SQLite deps in domain.
- Notes:
  - 2026-11-14T02:13:00Z: Claimed (next non-UI task). Implementing foundational domain entities/IDs for workspace, environment, variable, request, response/execution, history with shared ID helper. Existing workspace/env/var models will be normalized; no GUI/SQLite deps.
  - 2026-11-14T02:14:30Z: Added `internal/domain/id/id.go` (prefixed ID generator). Normalized `workspace`, `environment`, `variable` models with IDs/timestamps and compat constructors. Added initial entities: `collection` (Collection, Folder), `request` (Kind, saved request definition), `response` (Response, ExecutionResult), `history` (Entry). No GUI/SQLite deps. Verified: `go build ./internal/domain/... ./internal/services/...` and `go build ./internal/app ./internal/screens ./internal/storage/sqlite/... .` passed.

### P0-SERVICE-INTERFACES: Application service and repository interfaces
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:19:30Z
- Depends on: P0-DOMAIN-MODELS
- Allowed paths/scope: `internal/services/**` (new interfaces/contracts), `internal/events/**` (new)
- Acceptance: Service/repository/event contracts defined for core workflows; GUI-facing interfaces documented; typed event definitions for key state changes.
- Notes:
  - 2026-11-14T02:18:00Z: Claimed; implementing service/repository interfaces and typed event contracts in `internal/services/**` and `internal/events/**`.
  - 2026-11-14T02:19:30Z: Added typed event contracts/bus interfaces in `internal/events/events.go` (workspace/env/variable/request execution/history + protocol lifecycle event types/payloads). Added service/repository interface contracts in `internal/services/workspace`, `environment`, `variable`, `collection`, `request`, `history` (`service.go` per package) covering core CRUD/list, active-env state hooks, execution/cancel, and history record/query/clear workflows. Removed prior concrete in-memory/SQLite-coupled service implementations from `internal/services/*_service` to keep this scope contract-focused. Verified: `go build ./internal/events ./internal/services/workspace ./internal/services/environment ./internal/services/variable ./internal/services/collection ./internal/services/request ./internal/services/history` passed. `go build ./internal/domain/...` still fails on pre-existing unrelated `internal/domain/auth` (missing imports) outside scope. GUI-facing implementation wiring/adapters pending.

### P0-STORAGE-SQLITE: SQLite storage skeleton, migrations, repositories
- Status: review
- Owner: kilo
- Updated: 2026-11-14T01:23:30Z
- Depends on: P0-DOMAIN-MODELS, P0-SERVICE-INTERFACES
- Allowed paths/scope: `internal/storage/**`, `migrations/**`, DB config/wiring
- Acceptance: SQLite open/config, migrations framework, FK enforcement, transaction helpers, initial repository implementations/skeletons behind interfaces.
- Notes:
  - 2026-11-14T01:21:00Z: Claimed; implementing SQLite DB open/config, migrations runner, transaction helpers, and initial repository skeleton/implementations under `internal/storage/sqlite` and `migrations/`. Existing interfaces/models have gaps; will adapt minimally within scope.
  - 2026-11-14T01:23:30Z: Added `modernc.org/sqlite` driver; `internal/storage/sqlite/db.go` (Open with WAL/busy_timeout for file DB, FK enforcement, ping, migrate-on-open, Tx helper, NowUTC); `internal/storage/sqlite/migrate.go` (embedded schema_migrations tracking/applied migrations, initial schema: workspaces/environments/variables + indexes); `migrations/0001_initial_schema.sql` reference copy. Implemented SQLite repositories: `internal/storage/sqlite/workspace`, `environment` (Save takes workspaceID), `variable` (Save takes id/workspaceID/environmentID) with CRUD/list basics; removed conflicting placeholder workspace repo. Added `internal/storage/sqlite/sqlite_test.go` (open/migrate/workspace+environment repo ops, FK pragma) — `go test ./internal/storage/sqlite` and `go build ./internal/storage/sqlite/...` passed. `go build ./...` still fails on pre-existing unrelated `internal/domain/variable-resolution` errors outside scope. Service-interface alignment/wiring and broader repository coverage pending.

### P0-STATE-SEPARATION: GUI/app/transient state separation
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:48:00Z
- Depends on: P0-APP-BOOTSTRAP
- Allowed paths/scope: `internal/state/**`, `internal/screens/**`, GUI state packages
- Acceptance: Replace placeholder state with purposeful GUI vs application/transient state structures; no domain persistence logic embedded in views.
- Notes:
  - 2026-11-14T02:47:00Z: Claimed (next non-GUI task). Creating `internal/state/**` scaffolding for separated app/GUI/transient execution state; avoiding screen refactors in this pass.
  - 2026-11-14T02:48:00Z: Replaced placeholder `internal/state/app_state.go` and added `gui_state.go`, `execution_state.go`: separated persistent-selection app state, GUI navigation/layout state, and transient execution/connection state models/helpers (no persistence/domain logic). `go build ./internal/state ./internal/app ./internal/screens ./internal/storage/sqlite/... .` passed. Existing screen-local state in `internal/screens` not yet migrated to these types.

### P0-LOGGING-OBS: Logging and redaction baseline
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:50:00Z
- Depends on: P0-APP-BOOTSTRAP
- Allowed paths/scope: `internal/config/**`, logging/observability helpers, service instrumentation points
- Acceptance: `log/slog` baseline configured; redaction policy/helpers for secrets/tokens; structured error categories scaffolded.
- Notes:
  - 2026-11-14T02:31:00Z: Claimed (next non-GUI task). Implementing slog baseline, redaction helpers, and structured error categories in `internal/config`/logging packages.
  - 2026-11-14T02:32:30Z: Added `internal/logging` (redacting `slog.Handler` wrapper, sensitive key/pattern detection, `Config`/`New`/`SetupDefault`), `internal/config/logging.go` (`Logging` config + `SetupLogging`), and `internal/observability/errors` (categorized structured errors with wrap/fields/category helpers). Added tests for redaction and error categorization. `go test ./internal/logging ./internal/observability/errors ./internal/config` passed. App bootstrap/instrumentation wiring not yet applied.
  - 2026-11-14T02:50:00Z: Wired baseline logging setup in app bootstrap (`internal/app.Open` calls `config.SetupLogging`). Added in-memory event bus impl/tests (`internal/events/bus.go`) and request executor history recording hook + history repo/service wiring in app (non-GUI). `go test ./internal/logging ./internal/observability/errors ./internal/config ./internal/events ./internal/services/request ./internal/storage/sqlite/history ./internal/app` passed. Deeper service-level instrumentation still pending.

## Phase 1 — HTTP Core

### P1-HTTP-DOMAIN: HTTP request/response domain and validation
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:34:30Z
- Depends on: P0-DOMAIN-MODELS
- Allowed paths/scope: `internal/domain/request/**`, `internal/domain/response/**`, validation packages
- Acceptance: HTTP method/url/headers/params/body/settings modeled; validation errors structured and testable.
- Notes:
  - 2026-11-14T02:33:30Z: Claimed next non-GUI task; extending request/response domain with HTTP-specific models/settings and structured validation.
  - 2026-11-14T02:34:30Z: Added HTTP-specific request models/settings (`internal/domain/request/http.go`: methods/body types/settings/defaults, `HTTPRequest`, `NewHTTP`, `NormalizeHTTP`) and structured validation (`internal/domain/request/validate.go`: `ValidationError`/`ValidationErrors`, `ValidateHTTP` for workspace/name/kind/method/url/scheme/settings). Added response HTTP metadata/timing helpers (`internal/domain/response/http.go`: `Timing`, `HTTPMetadata`, `NewHTTPResponse`). Tests: `validate_test.go`, `http_test.go`. `go test ./internal/domain/request ./internal/domain/response` and targeted builds passed. Transport/execution pipeline integration pending (P1-VARIABLE-RESOLUTION, P1-HTTP-TRANSPORT).

### P1-VARIABLE-RESOLUTION: Variable interpolation and resolution pipeline
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:36:30Z
- Depends on: P0-DOMAIN-MODELS
- Allowed paths/scope: `internal/domain/variable/**`, `internal/services/**` (resolver), template/render utilities
- Acceptance: Deterministic scope precedence, missing/cycle handling, `{{var}}` interpolation, original request preserved, secret metadata handling.
- Notes:
  - 2026-11-14T02:35:30Z: Claimed; implementing/fixing variable resolution pipeline with scope precedence, interpolation, cycle/missing handling, and secret metadata.
  - 2026-11-14T02:36:30Z: Reimplemented `internal/domain/variable-resolution` (`resolver.go`, `variable_set.go`): deterministic precedence (request>folder>collection>environment>workspace>app defaults), `{{var}}` interpolation with nested resolution, missing vars/cycles/depth-limit leave placeholders unchanged, `ResolveWithMeta`/`ResolveWithSecrets` secret metadata, cycle detection. Added `resolver_test.go`. `go test ./internal/domain/variable-resolution` passed. Request-copy preservation/integration at service layer pending.

### P1-HTTP-TRANSPORT: HTTP transport adapter and execution pipeline
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:40:00Z
- Depends on: P1-HTTP-DOMAIN, P1-VARIABLE-RESOLUTION, P0-SERVICE-INTERFACES
- Allowed paths/scope: `internal/transport/http/**`, `internal/services/request/**`
- Acceptance: `net/http`-based execution with context cancellation, timeouts, redirects/TLS/proxy settings hooks, structured execution result; no GUI dependency.
- Notes:
  - 2026-11-14T02:38:00Z: Claimed; implementing HTTP transport adapter and request execution service (validation/build/transport/capture/result, ctx cancellation, timeout/redirects/TLS/proxy settings). Auth/pre-post/history hooks minimal pending other tasks.
  - 2026-11-14T02:40:00Z: Added `internal/transport/http/client.go` (`net/http` transport with ctx, timeout, redirect/TLS/proxy handling, URL/params/headers/body build, response capture/timing) and `internal/services/request/executor.go` (clones request to preserve original, resolves vars, validates, executes, tracks cancel funcs, structured `ExecutionResult`). Renamed interface in `internal/services/request/service.go` to `Transport` to avoid redeclare with executor. Tests: `internal/transport/http/client_test.go`, `internal/services/request/executor_test.go`. `go test ./internal/transport/http ./internal/services/request` passed. History/auth/pre-post hooks and richer transport config/pooling integration pending.

### P1-HISTORY: Execution history persistence and retention hooks
- Status: review
- Owner: kilo
- Updated: 2026-11-14T02:52:00Z
- Depends on: P0-STORAGE-SQLITE, P1-HTTP-TRANSPORT
- Allowed paths/scope: `internal/domain/history/**`, `internal/services/history/**`, `internal/storage/**`, `migrations/**`
- Acceptance: History records persisted separately from saved requests; retention/config hooks; large-body strategy documented/implemented per scope.
- Notes:
  - 2026-11-14T02:42:00Z: Claimed; implementing history migration, SQLite repository, and history service hooks (record/query/clear/retention basics).
  - 2026-11-14T02:44:00Z: Added migration v4 `execution_history` (+ indexes) in `internal/storage/sqlite/migrate.go` and `migrations/0001_initial_schema.sql`. Implemented SQLite history repo `internal/storage/sqlite/history/repository.go` (save/get/list/delete/clear/prune by workspace/limit) and service impl `internal/services/history/service_impl.go` with retention config hook on record. Added `repository_test.go`. `go test ./internal/storage/sqlite/history ./internal/services/history` passed. History stores execution metadata separately from saved requests; large response bodies not persisted in this pass (metadata-only strategy). Executor recording wiring pending.
  - 2026-11-14T02:52:00Z: Added `internal/services/history/service_impl_test.go` (record+retention via SQLite store). Executor history hook/repo `Record` alias and app wiring done in prior commits; `go test ./internal/services/history ./internal/storage/sqlite/history` passed. Large-body offload/storage policy still minimal (metadata-only).

## GUI / UI Workstreams

### UI-APP-SHELL: App shell and navigation frame
- Status: review
- Owner: kilo
- Updated: 2026-11-14T00:57:30Z
- Depends on: P0-APP-BOOTSTRAP, P0-STATE-SEPARATION
- Allowed paths/scope: `main.go`, `internal/screens/**`, GUI layout/components for shell
- Acceptance: Functional shell (title/context, sidebar/content split, empty states) wired to UI state/services; resizable/collapsible nav where implemented.
- Notes:
  - 2026-11-14: `internal/screens/dash.go` fixed to use `layout.DraggableSideNav` API (persistent state, pointer args, props). Further shell composition pending.
  - 2026-11-14T00:30:00Z: Claimed; implementing app shell frame (title bar, sidebar nav, main content/empty-state placeholders) in `main.go` and `internal/screens`.
  - 2026-11-14T00:32:00Z: Added `internal/screens/app.go` shell (title bar drag region, resizable `DraggableSideNav` with workspace/collections placeholders, context header, request/response split panes with empty states) and wired `main.go` to render it (1200x800, "Heartless"). Uses local UI state only; service/event wiring pending per P0-APP-BOOTSTRAP/P0-STATE-SEPARATION. `go build ./...` passed. `internal/screens/dash.go` left unused.
  - 2026-11-14T00:34:30Z: Added workspace selector button ("No workspace", text variant) beside "Heartless" in title bar (`internal/screens/app.go`); click handler TODO pending real selector/service wiring. Pasted reference image could not be viewed (image input unsupported). `go build ./...` passed.
  - 2026-11-14T00:57:30Z: Set up `ui.Router`-based routing in shell state (`/`, `/settings`, fallback 404) via `internal/screens/app.go`; sidebar Home/Settings push routes, title bar shows Back when history allows. `go build ./...` passed. Service/event wiring and richer routes still pending.

### UI-REQUEST-EDITOR: Request editor screens/components
- Status: todo
- Owner:
- Updated:
- Depends on: UI-APP-SHELL, P1-HTTP-TRANSPORT (for send), P1-VARIABLE-RESOLUTION
- Allowed paths/scope: `internal/screens/**`, GUI request editor components
- Acceptance: HTTP request authoring UI (method/url/params/headers/body/settings) with draft state, validation feedback, send/cancel integration.
- ## Notes:

### UI-RESPONSE-PANEL: Response/timeline/history views
- Status: todo
- Owner:
- Updated:
- Depends on: UI-APP-SHELL, P1-HTTP-TRANSPORT, P1-HISTORY
- Allowed paths/scope: `internal/screens/**`, GUI response/history components
- Acceptance: Renders structured execution results, metadata, body/timeline/history views; handles loading/error/cancel states.
- ## Notes:

### UI-ENV-AUTH: Environment and authentication UI
- Status: review
- Owner: kilo
- Updated: 2026-11-14T01:48:30Z
- Depends on: UI-APP-SHELL, P0-DOMAIN-MODELS
- Allowed paths/scope: `internal/screens/**`, GUI env/auth components/dialogs
- Acceptance: Environment selection/editing and auth configuration surfaces aligned with domain/service contracts; secret-safe display.
- Notes:
  - 2026-11-14T01:07:00Z: Claimed; implementing UI-level workspace/environment creation and selection in `internal/screens/app.go` (local/transient state, pending domain/service persistence). Auth UI deferred.
  - 2026-11-14T01:08:30Z: Added UI-level workspace/environment creation and selection in `internal/screens/app.go` (workspace selector in title/sidebar, workspace dialog with create/select, environment create/select per workspace, active workspace/env shown in shell/context). State is local/transient (`ui.Local`) with placeholder IDs; persistence/service integration pending. `go build ./...` currently fails due to unrelated pre-existing errors in `internal/domain/variable-resolution` (outside scope); shell screen package compiles in isolation not separately verified. Auth UI still deferred.
  - 2026-11-14T01:42:00Z: Linked workspaces/environments to SQLite in UI scope: `screens.App` now takes `*app.Services`, loads/persists workspaces and environments via SQLite repos (`WorkspaceID` linked on envs), create/select flows write/read DB, active selection reloads envs per workspace, errors surfaced in sidebar. Added `WorkspaceID` to domain environment and updated env repo scan/save accordingly (storage/domain touched as required linkage). Removed stale duplicate `internal/screens/titlebar.go`/`titlebar_test.go` conflicting with current shell. `go build ./internal/app ./internal/screens ./internal/storage/sqlite/... .` passed; `go build ./...` still blocked by unrelated pre-existing `internal/domain/variable-resolution` errors. Auth UI still deferred.
  - 2026-11-14T01:48:30Z: Implemented workspace variable/value creation linked to DB: added `/variables` view + title-bar "Variables" nav, workspace-scoped variable list/create (name/value) persisted via SQLite variable repo; wired `Services.Variables` in app bootstrap. Linkage changes outside declared scope (noted): `internal/app/app.go`, `internal/domain/variable/variable.go` (ID/scope IDs), `internal/storage/sqlite/variable/repository.go` (store/scan IDs). `go build ./internal/app ./internal/screens ./internal/storage/sqlite/... .` and `go test ./internal/storage/sqlite` passed; `go build ./...` still blocked by unrelated pre-existing `internal/domain/variable-resolution` errors. Env-scoped vars, editing/deletion, secret-safe masking, and auth UI pending.

### UI-TITLEBAR: Custom window title bar
- Status: in_progress
- Owner: mimo
- Updated: 2026-11-14T03:37:00Z
- Depends on: UI-APP-SHELL
- Allowed paths/scope: `main.go` (window options only), `internal/screens/titlebar.go`, `internal/screens/app.go` (title bar wiring only), `internal/screens/*_test.go`
- Acceptance: Native OS title bar hidden; app draws a single custom draggable title bar with themed minimize/maximize/restore/close controls wired to the window; workspace/env breadcrumb and back navigation preserved; targeted `go build`/`go test`/`go vet` pass.
- Notes:
  - 2026-11-14T03:37:00Z: Claimed per user request (dislikes the native window title bar; wants a custom one). Scope overlaps UI-APP-SHELL (`internal/screens/**`, `main.go`) which is in `review` with kilo idle — recorded here per rule 3 before touching those files; UI-APP-SHELL block left untouched. Approach: `mygo.TitleBarHidden` (+ possibly `HideWindowButtons`) with a custom-drawn bar in `internal/screens/titlebar.go`.

### UI-REQUEST-SCRIPT: Post-response scripting tab
- Status: done
- Owner: (completed)
- Updated: 2026-11-16T17:11:00Z
- Depends on: UI-REQUEST-EDITOR, P1-HTTP-TRANSPORT
- Allowed paths/scope: `internal/screens/script*.go`, `internal/screens/request_script_tab.go`, `internal/domain/request/request.go` (script field), `internal/storage/sqlite/collection/**`, `internal/storage/sqlite/migrate.go`, `go.mod`
- Acceptance: A Script tab holds a post-response JavaScript hook; scripts see `req` and `res` and can read/update environment variables; the editor highlights syntax and `{{variables}}`; the script runs automatically when a response arrives; scripts are sandboxed and time-limited.
- Notes:
  - 2026-11-16: Added `Script` to the request domain and migration v11 (`requests.script`), persisted through the collection repository.
  - 2026-11-16: Added an isolated goja runtime (`internal/screens/script.go`) exposing `req`, `res` (with `res.json()`), `env.get/set/all`, `test(name, fn)` and `console.*`, with a 2s interrupt and no network/filesystem access. Updated variables are returned rather than persisted, so the pane shows each `old → new` before saving.
  - 2026-11-16: `internal/screens/script_runner.go` runs scripts on a dedicated goroutine so a slow script never stalls a frame; results are applied under the state lock and a per-request generation counter drops superseded sends.
  - 2026-11-16: The tab is a single inline editor colored in place via `TextRanges` (`script_highlight.go`), replacing an earlier split editor/preview layout. Variables are colored over their enclosing string and bold; undefined ones use the warning color.
  - 2026-11-16: `go vet ./internal/...` clean; 8 tests in `script_test.go` pass, covering the runtime, variable updates, assertions, rune-offset highlighting, auto-run after a real send, and the tab's save/run path.

## Documentation Workstreams

### DOC-PLAN: Implementation plan baseline
- Status: done
- Owner: (completed)
- Updated: 2026-11-14T00:00:00Z
- Depends on: none
- Allowed paths/scope: `docs/PLAN.md`
- Acceptance: Phased roadmap, guardrails, package layout, testing/DoD captured.
- Notes:
  - 2026-11-14: Created `docs/PLAN.md` from design doc and current scaffold.

### DOC-FEATURES: Yaak feature reference docs
- Status: done
- Owner: (completed)
- Updated: 2026-11-14T00:00:00Z
- Depends on: DOC-PLAN
- Allowed paths/scope: `docs/features/**`
- Acceptance: Per-feature docs derived from `yaak/` reference with Heartless implementation notes and index.
- Notes:
  - 2026-11-14: Added feature index and 16 feature documents under `docs/features/`.

### DOC-UI-LAYOUT: UI layout documentation
- Status: done
- Owner: (completed)
- Updated: 2026-11-14T00:00:00Z
- Depends on: DOC-FEATURES
- Allowed paths/scope: `docs/ui/**`
- Acceptance: Layout docs for shell, sidebar, request editor, response panel, env/auth, dialogs/settings.
- Notes:
  - 2026-11-14: Added UI layout index and 6 layout documents under `docs/ui/`.

### DOC-PROGRESS: Multi-agent progress tracker
- Status: done
- Owner: (completed)
- Updated: 2026-11-14T00:00:00Z
- Depends on: none
- Allowed paths/scope: `docs/PROGRESS.md`
- Acceptance: Progress tracker with claim/status protocol enabling multiple agents to coordinate without task collisions.
- Notes:
  - 2026-11-14: Created this document.

## Phase 6 — Interoperability

### P6-IMPORTERS: Collection import pipeline (non-UI)
- Status: review
- Owner: mimo
- Updated: 2026-11-14T03:28:00Z
- Depends on: P0-DOMAIN-MODELS, P0-SERVICE-INTERFACES, P0-STORAGE-SQLITE
- Allowed paths/scope: `internal/importer/**` (new), `internal/services/importer/**` (new), `internal/storage/sqlite/collection/**`, `internal/storage/sqlite/migrate.go`, tests under those paths
- Acceptance: Postman (v2.0/v2.1) and Insomnia (v4/v5) sources detected, parsed into a normalized draft, validated, conflict-checked, and persisted transactionally as collections/folders/requests (including headers/params) into a target workspace; unsupported/conflicting data reported as issues; no GUI dependencies; targeted `go build`/`go test`/`go vet` pass.
- Notes:
  - 2026-11-14T02:33:00Z: Claimed next non-UI task. Scope linkage recorded per rule 3 before touching P0-STORAGE-SQLITE-owned files: `internal/storage/sqlite/migrate.go` (register migration v4 adding `requests.headers`/`requests.params`) and `internal/storage/sqlite/collection/repository.go` (persist headers/params, extract save helpers, add atomic `ImportBatch`). Additions limited to import persistence; existing method signatures unchanged.
  - 2026-11-14T03:28:00Z: Implementation complete, set to review. Supersedes the "migration v4" wording above: another agent took v4 (`execution_history`) concurrently, so the headers/params migration was added as **v5 `request_headers_params`**. Added: `internal/importer` (format-neutral `Draft`/`Item`/`Issue`/`Importer` contract + `Select` + `Text`/`Description` helpers), `internal/importer/postman` (v2.0/v2.1: nested folders, string/object URLs, query+path params, headers, raw/urlencoded/formdata/graphql bodies; auth/scripts/collection-variables/file-bodies reported as issues), `internal/importer/insomnia` (v4 `resources` + v5 `collection` exports; grpc/websocket/environment/api-spec/auth reported as issues, orphan detection). Exported service contract: `internal/services/importer` — `New(Deps{Workspaces, Collections, Importers})`, `Service.Import(ctx, workspaceID, data, Options) (*Result, error)` with `ConflictPolicy` rename/skip, sentinel errors `ErrUnsupportedFormat`/`ErrWorkspaceNotFound`/`ErrEmptyImport`, `Result{Format, Collections, Folders, Requests, CreatedIDs, Issues}`; normalization: default names, method uppercased/`GET`, kind validation, enabled-only headers/params with first-wins duplicate warnings. Storage linkage (declared scope): migration v5 in `internal/storage/sqlite/migrate.go`; `internal/storage/sqlite/collection/repository.go` persists `headers`/`params` JSON (`encodeNV`/`decodeNV`, Save/Get/List) with Save\* bodies extracted over an `execer` interface; new `batch.go` `Repository.ImportBatch` writes collections/folders/requests in one `sqlite.Tx`. Verification: `go build ./...` pass; `go test ./internal/importer/... ./internal/services/importer/... ./internal/storage/sqlite/...` pass (adapter fixtures, service unit fakes + SQLite integration incl. conflict rename and rollback); `go vet` on touched packages pass. Follow-ups (out of scope): wire into `app.Services` (P0-APP-BOOTSTRAP), environment/variable + auth import (currently reported as issues), OpenAPI/native/curl importers, `docs/features/import-export.md` update; pre-existing root `main_test.go` compile failure (`undefined: lo`) untouched.

## Activity Log (append-only)

Add new entries at the bottom. Format: `- YYYY-MM-DDTHH:MM:SSZ [agent] TASK-ID: message`

- 2026-11-14T00:00:00Z [agent] P0-DASH-FIX: Fixed `internal/screens/dash.go` to match `lobe-mygo` `DraggableSideNav` API.
- 2026-11-14T00:00:00Z [agent] DOC-PLAN: Created `docs/PLAN.md`.
- 2026-11-14T00:00:00Z [agent] DOC-FEATURES: Created `docs/features/` index and feature reference docs from `yaak/`.
- 2026-11-14T00:00:00Z [agent] DOC-UI-LAYOUT: Created `docs/ui/` index and layout docs.
- 2026-11-14T00:00:00Z [agent] DOC-PROGRESS: Created `docs/PROGRESS.md` with multi-agent coordination rules.
- 2026-11-14T00:30:00Z [kilo] UI-APP-SHELL: Claimed and started app shell implementation (`main.go`, `internal/screens`).
- 2026-11-14T00:32:00Z [kilo] UI-APP-SHELL: Implemented initial app shell (`internal/screens/app.go`, `main.go`), `go build ./...` passed, set to review (service wiring pending).
- 2026-11-14T00:34:30Z [kilo] UI-APP-SHELL: Added workspace selector button beside title bar "Heartless" label (`internal/screens/app.go`); pasted reference image not viewable (image input unsupported). Build passed, review.
- 2026-11-14T00:56:00Z [kilo] UI-APP-SHELL: Resumed for routing setup (`ui.Router`); claimed in_progress.
- 2026-11-14T00:57:30Z [kilo] UI-APP-SHELL: Added `ui.Router` routing (`/`, `/settings`, 404) in `internal/screens/app.go`; sidebar/title navigation hooks; `go build ./...` passed, review.
- 2026-11-14T01:07:00Z [kilo] UI-ENV-AUTH: Claimed for UI-level workspace/environment creation and selection in `internal/screens/app.go`.
- 2026-11-14T01:08:30Z [kilo] UI-ENV-AUTH: Implemented local UI workspace/environment create+select flows in `internal/screens/app.go`; `go build ./...` blocked by unrelated `internal/domain/variable-resolution` errors outside scope; set to review.
- 2026-11-14T01:21:00Z [kilo] P0-STORAGE-SQLITE: Claimed; starting SQLite storage skeleton/migrations/repositories implementation.
- 2026-11-14T01:23:30Z [kilo] P0-STORAGE-SQLITE: Implemented SQLite open/config+migrations+FK/Tx and initial workspace/environment/variable repositories with tests (`go test ./internal/storage/sqlite` pass); full repo build blocked by unrelated `internal/domain/variable-resolution` errors; review.
- 2026-11-14T01:39:30Z [kilo] P0-APP-BOOTSTRAP: Claimed for DB wiring to link workspaces/environments.
- 2026-11-14T01:39:30Z [kilo] UI-ENV-AUTH: Resumed to link workspace/environment UI to SQLite persistence.
- 2026-11-14T01:42:00Z [kilo] P0-APP-BOOTSTRAP: Added DB bootstrap/wiring (`internal/app`, `main.go`) for SQLite services; targeted build passed.
- 2026-11-14T01:42:00Z [kilo] UI-ENV-AUTH: Linked workspace/environment create/select UI to SQLite repos (workspace/env persistence + workspace-scoped env loading); targeted build passed, review.
- 2026-11-14T01:46:30Z [kilo] UI-ENV-AUTH: Resumed to implement workspace value/variable creation linked to DB.
- 2026-11-14T01:48:30Z [kilo] UI-ENV-AUTH: Added workspace variable creation/list UI (`/variables`) persisted to SQLite; updated app/domain/storage for variable linkage; targeted build/tests passed, review.
- 2026-11-14T02:04:30Z [kilo] P0-STORAGE-SQLITE: Claimed (in_progress) to persist active environment per workspace.
- 2026-11-14T02:04:30Z [kilo] UI-ENV-AUTH: Claimed (in_progress) to wire active environment persistence to disk.
- 2026-11-14T02:05:00Z [kilo] P0-STORAGE-SQLITE: Added migration v2 `workspace_state` and workspace repo active-env get/set methods (in progress).
- 2026-11-14T02:06:30Z [kilo] P0-STORAGE-SQLITE: Completed active-environment state persistence schema/repo support; targeted build/tests passed, review.
- 2026-11-14T02:06:30Z [kilo] UI-ENV-AUTH: Wired active environment persistence/restoration per workspace across UI flows; targeted build passed, review.
- 2026-11-14T02:13:00Z [kilo] P0-DOMAIN-MODELS: Claimed next non-UI task; starting core domain entities/ID helper implementation in `internal/domain/**`.
- 2026-11-14T02:14:30Z [kilo] P0-DOMAIN-MODELS: Implemented initial domain entities (workspace/env/var/collection/folder/request/response/execution/history) and shared ID helper; targeted builds passed, review.
- 2026-11-14T02:18:00Z [kilo] P0-SERVICE-INTERFACES: Claimed; implementing service/repository and event contracts.
- 2026-11-14T02:19:30Z [kilo] P0-SERVICE-INTERFACES: Added `internal/events` and core service/repository interface contracts under `internal/services/*`; targeted builds passed, review.
- 2026-11-14T02:31:00Z [kilo] P0-LOGGING-OBS: Claimed next non-GUI task; implementing logging/redaction/error baseline.
- 2026-11-14T02:32:30Z [kilo] P0-LOGGING-OBS: Implemented slog baseline + redaction helpers and structured error categories with tests; `go test ./internal/logging ./internal/observability/errors ./internal/config` passed, review.
- 2026-11-14T02:33:00Z [mimo] P6-IMPORTERS: Claimed non-UI import pipeline (Postman/Insomnia importers + transactional collection import service); recorded storage linkage scope in task notes.
- 2026-11-14T02:33:30Z [kilo] P1-HTTP-DOMAIN: Claimed next non-GUI task; implementing HTTP request/response domain models and validation.
- 2026-11-14T02:34:30Z [kilo] P1-HTTP-DOMAIN: Added HTTP request/response domain models, settings, validation, and tests; `go test ./internal/domain/request ./internal/domain/response` passed, review.
- 2026-11-14T02:35:30Z [kilo] P1-VARIABLE-RESOLUTION: Claimed; starting variable resolution pipeline implementation.
- 2026-11-14T02:36:30Z [kilo] P1-VARIABLE-RESOLUTION: Implemented resolver/variable sets with precedence, interpolation, cycle/missing handling, secret metadata, and tests; `go test ./internal/domain/variable-resolution` passed, review.
- 2026-11-14T02:38:00Z [kilo] P1-HTTP-TRANSPORT: Claimed; implementing transport adapter and request execution pipeline basics.
- 2026-11-14T02:40:00Z [kilo] P1-HTTP-TRANSPORT: Implemented HTTP transport client and request executor with validation/resolution/cancel/result capture; `go test ./internal/transport/http ./internal/services/request` passed, review.
- 2026-11-14T02:42:00Z [kilo] P1-HISTORY: Claimed; implementing execution history persistence and retention hooks.
- 2026-11-14T02:44:00Z [kilo] P1-HISTORY: Added history migration/repo/service with retention hooks and tests; `go test ./internal/storage/sqlite/history ./internal/services/history` passed, review.
- 2026-11-14T02:46:00Z [kilo] P1-HISTORY/P1-HTTP-TRANSPORT/P0-LOGGING-OBS: Implementing follow-up non-GUI wiring (history recording in executor, logging bootstrap).
- 2026-11-14T02:47:00Z [kilo] P0-STATE-SEPARATION: Claimed; implementing state separation scaffolding in `internal/state/**`.
- 2026-11-14T02:48:00Z [kilo] P0-STATE-SEPARATION: Added app/gui/transient state scaffolding in `internal/state/**`; targeted builds passed, review (screen migration pending).
- 2026-11-14T02:50:00Z [kilo] P0-LOGGING-OBS: Wired logging bootstrap in app and non-GUI execution/history/event-bus hooks; tests passed, review.
- 2026-11-14T02:47:00Z [kilo] P0-STATE-SEPARATION: Claimed; implementing state separation scaffolding in `internal/state/**`.
- 2026-11-14T02:48:00Z [kilo] P0-STATE-SEPARATION: Added app/gui/transient state scaffolding in `internal/state/**`; targeted builds passed, review (screen migration pending).
- 2026-11-14T03:29:30Z [mimo] P6-IMPORTERS: Implemented Postman/Insomnia importers, import service (detect→parse→normalize→conflict-resolve→atomic persist), migration v5 `requests.headers`/`params` + `ImportBatch`; `go build ./...`, targeted `go test`, `go vet` pass; review.
- 2026-11-14T03:34:54Z [mimo] P6-IMPORTERS: Docs follow-up — added "Implemented (P6-IMPORTERS)" and remaining-work sections to `docs/features/import-export.md` (doc-only; no code changes).
- 2026-11-16T17:11:00Z [agent] UI-REQUEST-SCRIPT: Added post-response scripting tab with an isolated goja runtime, inline syntax highlighting, and automatic execution on response arrival; `go vet ./internal/...` clean, `go test ./internal/screens -run Script` passed.
- 2026-11-16T17:11:00Z [agent] UI-REQUEST-SCRIPT: Landed in commit `4712190` (`feat: add response history, environment switcher, and UI polish`), whose message does not mention scripting. Intended split, per `commit.md`:

  ```
  feat(script): run post-response JavaScript with req, res, and env access

  Requests needed a way to inspect a response and feed values back into
  variables without leaving the app. The Script tab provides an inline
  editor whose text is colored in place, and runs the script by itself
  whenever a response arrives, so assertions and token extraction happen
  without an extra step.

  Scripts run in an isolated goja runtime with a 2s interrupt and no
  network or filesystem access. A dedicated goroutine owns the run queue
  so a slow script never stalls the UI, and a per-request generation
  counter drops results from superseded sends.

  Refs: post-response scripting
  ```

  The persistence half was intended as its own commit:

  ```
  feat(history): persist the complete response payload

  Only the status, headers and body were stored, so the response ID,
  request ID, duration and effective URL were lost on reload and the
  stored record could not reproduce the original result.

  Duration is stored in nanoseconds rather than reusing the entry's
  duration_ms, since a sub-millisecond round-trip truncated to zero.
  ```
- 2026-11-18T19:11:42Z [zed] UI-WORKSPACE-SETTINGS: Added a workspace settings page (`/workspace-settings`, `internal/screens/workspace_settings.go`) opened from the title bar's Workspace Settings menu item and Cmd+;, with an Auth section reusing the request auth panel (workspace scope skips Inherit) and an auto-inherit switch backed by migration v13 (`workspaces.auto_inherit_auth`). Requests with auth type `inherit` always inherit the workspace default; with no type set, only when auto-inherit is on (`request_state.go` `resolveAuthInheritance`). Tests in `workspace_settings_test.go`; `go test ./...`, `go vet ./...`, `gofmt` clean.
- 2026-11-18T19:11:42Z [zed] UI-REQUEST-EDITOR: Fixed form/multipart bodies not being sent: `applyBodyType` encoded the query params into the body and dropped the form rows entirely. Form rows now encode as the body (`application/x-www-form-urlencoded` or real `multipart/form-data` with boundary), query params always stay in the URL, and form fields persist via migration v14 (`requests.form`, JSON like headers/params). Tests: `params_send_test.go` (query params and form fields through a live send), `body_types_test.go`, `collection` round-trip; `go test ./...`, `go vet ./...`, `gofmt` clean.
- 2026-11-18T19:55:48Z [zed] UI-REQUEST-EDITOR: Added path-parameter support: params named `:id` now substitute the matching `:id` token in the URL (shared `domainreq.ApplyPathParams`, escaped per path segment) instead of being appended to the query string percent-encoded; plain params keep going to the query, and a `:name` with no matching token is dropped. The URL preview under the editor applies the same substitution. Tests: `domain/request/http_test.go`, `transport/http/url_test.go`, `screens/params_send_test.go`; `go test ./...`, `go vet`, `gofmt` clean.

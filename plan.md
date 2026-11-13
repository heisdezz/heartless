# Design Document — Yaak API Client

## 1. Project Overview

Build a native, cross-platform API development and testing client inspired by Yaak.

The application should provide a comprehensive environment for designing, organizing, executing, debugging, and testing HTTP requests and other API protocols.

The primary objective is to deliver a fast, reliable, extensible API client with a native Go architecture.

The application should prioritize:

- Performance and low resource consumption.
- Native execution without requiring a browser runtime.
- Extensible request and protocol architecture.
- Reliable collection and environment management.
- Reproducible API testing.
- Secure credential and secret management.
- Portable project data.
- Maintainable, modular Go code.

This is an engineering design document. It defines system behavior, architecture, responsibilities, and implementation constraints rather than visual appearance or UI specifications.

## 2. Core Design Principles

### 2.1 Native-first architecture

The application must be implemented primarily in Go.

Core functionality must not depend on JavaScript, Node.js, Electron, or browser APIs.

All essential operations, including request execution, serialization, authentication, storage, and environment resolution, must be implemented natively.

### 2.2 Separation of concerns

Business logic must remain independent of the GUI framework.

The GUI should communicate with application services through well-defined interfaces rather than directly manipulating internal application state.

### 2.3 Modular architecture

Each major subsystem must have a clearly defined responsibility.

Features should be independently testable and replaceable without requiring modifications throughout the application.

### 2.4 Extensibility

The architecture must accommodate additional protocols, authentication methods, scripting engines, storage backends, and import/export formats.

New capabilities should integrate through interfaces rather than tightly coupled implementations.

### 2.5 Data portability

User collections, environments, requests, and other project data should be exportable and portable.

The application should avoid unnecessary vendor-specific data formats.

---

## 3. Technology Stack

| Component            | Technology                         |
| -------------------- | ---------------------------------- |
| Programming language | Go                                 |
| GUI framework        | MyGO                               |
| UI component library | Lobe MyGO                          |
| HTTP execution       | Go `net/http`                      |
| WebSocket            | Native Go WebSocket implementation |
| GraphQL              | HTTP-based GraphQL transport       |
| Database             | SQLite                             |
| Database access      | Go database abstraction            |
| Serialization        | `encoding/json`                    |
| Configuration        | Go-native configuration            |
| Logging              | `log/slog`                         |
| Testing              | Go testing package                 |

Technology choices should remain behind interfaces wherever practical to avoid unnecessary coupling.

---

## 4. High-Level Architecture

The application follows a layered architecture.

```text
┌──────────────────────────────────────────┐
│               Presentation               │
│                                          │
│              MyGO + Lobe MyGO            │
└────────────────────┬─────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────┐
│              Application Layer           │
│                                          │
│  Request Service    Collection Service   │
│  Environment        History Service      │
│  Import / Export    Workspace Service    │
└────────────────────┬─────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────┐
│                 Domain Layer             │
│                                          │
│  Requests          Responses             │
│  Collections       Environments          │
│  Authentication    Variables             │
│  Assertions        Scripts               │
│  Protocol Models   Workspace             │
└────────────────────┬─────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────┐
│              Infrastructure              │
│                                          │
│  HTTP Transport    WebSocket Transport   │
│  SQLite            File Storage          │
│  Secret Storage    Import / Export       │
│  Logging           External Integrations │
└──────────────────────────────────────────┘
```

Dependencies should flow inward through interfaces. The domain layer must not depend on MyGO, SQLite, or concrete transport implementations.

---

## 5. Project Structure

Organize the project by domain and responsibility rather than placing all functionality into a small number of large packages.

```text
.
├── cmd/
│   └── yaak/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── bootstrap.go
│   │   └── lifecycle.go
│   │
│   ├── domain/
│   │   ├── request/
│   │   ├── response/
│   │   ├── collection/
│   │   ├── environment/
│   │   ├── workspace/
│   │   ├── authentication/
│   │   ├── history/
│   │   ├── assertion/
│   │   └── variable/
│   │
│   ├── services/
│   │   ├── request/
│   │   ├── collection/
│   │   ├── environment/
│   │   ├── history/
│   │   ├── workspace/
│   │   ├── importer/
│   │   └── exporter/
│   │
│   ├── transport/
│   │   ├── http/
│   │   ├── websocket/
│   │   ├── graphql/
│   │   └── grpc/
│   │
│   ├── auth/
│   │   ├── basic/
│   │   ├── bearer/
│   │   ├── apikey/
│   │   ├── oauth2/
│   │   └── digest/
│   │
│   ├── scripting/
│   │   ├── engine.go
│   │   ├── runtime/
│   │   └── execution/
│   │
│   ├── storage/
│   │   ├── sqlite/
│   │   ├── filesystem/
│   │   └── secrets/
│   │
│   ├── importer/
│   │   ├── postman/
│   │   ├── insomnia/
│   │   └── openapi/
│   │
│   ├── exporter/
│   │
│   ├── events/
│   │
│   ├── config/
│   │
│   └── platform/
│
├── gui/
│   ├── application/
│   ├── components/
│   ├── pages/
│   ├── state/
│   └── resources/
│
├── migrations/
│
├── tests/
│   ├── integration/
│   └── fixtures/
│
├── go.mod
├── go.sum
└── design.md
```

The structure is a starting point, not a requirement to create every package immediately. Introduce packages when their responsibilities become necessary.

---

## 6. Domain Design

### 6.1 Requests

A request represents an API operation that can be saved, modified, executed, and reused.

A request should support:

- HTTP method.
- URL.
- Query parameters.
- Headers.
- Path variables.
- Request body.
- Authentication configuration.
- Request timeout.
- Redirect policy.
- TLS configuration.
- Proxy configuration.
- Pre-request processing.
- Post-response processing.

Request definitions must be independent of their execution state.

A saved request should not contain transient response data or execution-specific information.

### 6.2 Responses

A response represents the result of executing a request.

It should capture:

- Status code.
- Status text.
- Response headers.
- Response body.
- Content type.
- Response size.
- Total execution duration.
- DNS lookup duration.
- Connection duration.
- TLS handshake duration.
- Time to first byte.
- Download duration.
- Effective URL.
- Redirect information.

Response bodies should support both textual and binary content.

Large response bodies should be handled through streaming or temporary storage instead of unnecessarily loading everything into memory.

### 6.3 Collections

Collections organize related API requests.

A collection should support:

- Nested folders.
- Saved requests.
- Collection-level authentication.
- Collection-level variables.
- Reusable configuration.
- Ordering.
- Descriptions.
- Metadata.

Collection inheritance should allow requests to inherit configuration from their parent collection while preserving the ability to override individual settings.

### 6.4 Environments

Environments provide reusable variables for different deployment targets.

Examples include:

- Development.
- Staging.
- Production.
- Local environments.

Variables should support:

- Key.
- Value.
- Type.
- Description.
- Secret designation.
- Scope.

The environment resolver should support variable interpolation in URLs, headers, query parameters, authentication, and request bodies.

Variable resolution should be deterministic and support clearly defined precedence:

1. Request-level variables.
2. Folder-level variables.
3. Collection-level variables.
4. Active environment variables.
5. Workspace-level variables.
6. Application defaults.

Secrets must not be exposed in logs or accidentally included in exports.

---

## 7. Request Execution Engine

The request execution engine is the central subsystem.

It should be independent of the GUI and callable through an application service.

### 7.1 Execution pipeline

```text
Request Definition
       │
       ▼
Validate Request
       │
       ▼
Resolve Variables
       │
       ▼
Apply Authentication
       │
       ▼
Run Pre-request Processing
       │
       ▼
Build HTTP Request
       │
       ▼
Configure Transport
       │
       ▼
Execute Request
       │
       ▼
Capture Response
       │
       ▼
Run Assertions / Post-processing
       │
       ▼
Persist Execution History
       │
       ▼
Return Execution Result
```

Each stage should be independently testable.

### 7.2 HTTP transport

Use Go's `net/http` package as the foundation.

The transport layer should support:

- HTTP/1.1.
- HTTP/2.
- TLS.
- Custom certificate authorities.
- Client certificates.
- Redirect configuration.
- Proxy configuration.
- Connection pooling.
- Request cancellation.
- Custom DNS configuration where supported.
- Configurable timeouts.
- Streaming uploads and downloads.

Transport configuration should be reusable across requests where appropriate.

Avoid creating a completely new HTTP transport for every request. Manage connection pools according to configuration and lifecycle.

### 7.3 Cancellation

Every execution must accept a `context.Context`.

Cancellation should propagate through:

- Request construction.
- Network execution.
- Body streaming.
- Post-processing.
- Persistence operations where appropriate.

The application must not leave abandoned goroutines or network operations running after cancellation.

### 7.4 Execution results

Execution should return a structured result rather than modifying the original request.

Conceptually:

```go
type ExecutionResult struct {
    RequestID string
    Response  *Response
    Duration  time.Duration
    Error     error
}
```

The actual implementation may include additional execution metadata.

---

## 8. Authentication System

Authentication should use a pluggable provider architecture.

Each authentication provider should be responsible for applying its credentials to an outgoing request.

Initial authentication methods:

- No authentication.
- Basic authentication.
- Bearer token.
- API key.
- OAuth 2.0.
- Digest authentication.
- Custom headers.

Authentication should support inheritance from collections and folders.

Credentials should be stored separately from ordinary request metadata.

Sensitive credentials should use the operating system's secure credential storage where available, with an appropriate encrypted fallback when necessary.

OAuth 2.0 token refresh should be handled by a dedicated service.

---

## 9. Variable Resolution

Variable resolution should be implemented as a standalone subsystem.

It must not be scattered throughout request construction or individual authentication providers.

The resolver should:

1. Identify variable references.
2. Determine their scope.
3. Resolve the effective value.
4. Handle missing variables according to an explicit policy.
5. Preserve secret metadata.
6. Produce a resolved request representation.

Support a consistent interpolation syntax:

```text
{{base_url}}
{{api_token}}
{{user_id}}
```

The resolver should avoid unintended recursive expansion and should detect circular references.

The original saved request must remain unchanged after resolution.

---

## 10. Scripting and Extensibility

Scripting should be isolated from the core execution engine.

The application must remain fully functional without requiring JavaScript.

### 10.1 Runtime abstraction

Define a runtime interface that can accommodate different execution engines.

```go
type Engine interface {
    Execute(
        ctx context.Context,
        source string,
        input ExecutionContext,
    ) (ExecutionOutput, error)
}
```

The interface should not expose runtime-specific implementation details to the application layer.

### 10.2 Goja integration

If JavaScript compatibility is required, Goja should be implemented as an isolated runtime adapter.

It must not become a dependency of request execution, storage, or other core services.

The scripting subsystem should support:

- Pre-request scripts.
- Post-response scripts.
- Environment variable updates.
- Request variable updates.
- Response inspection.
- Assertions.
- Execution time limits.
- Controlled access to application capabilities.

Script execution must be isolated from unrestricted filesystem, network, and process access.

Runtime errors should be returned as structured errors without crashing the application.

### 10.3 Assertions

Assertions should be a separate capability from scripting.

Provide native assertion functionality for common conditions:

- Status code.
- Response headers.
- Response body.
- JSON field values.
- Response duration.

This ensures basic testing functionality does not depend on JavaScript.

---

## 11. Protocol Architecture

Protocols should be implemented through independent transport adapters.

### HTTP

The initial and primary protocol.

Support all standard HTTP methods, custom methods, headers, query parameters, and request bodies.

### GraphQL

Build GraphQL on top of the HTTP transport.

Support:

- Queries.
- Mutations.
- Variables.
- Custom headers.
- GraphQL error inspection.

### WebSocket

Provide a persistent connection abstraction.

Support:

- Connection lifecycle.
- Sending messages.
- Receiving messages.
- Connection headers.
- Authentication.
- Reconnection controls.

### gRPC

Treat gRPC as a separate protocol adapter.

Support can be introduced incrementally, including service discovery, unary calls, and streaming operations.

Protocol-specific functionality must not leak into the generic request domain model.

---

## 12. Storage Architecture

SQLite should serve as the primary local persistence layer.

The storage implementation must be hidden behind repository interfaces.

### 12.1 Persistent data

Store:

- Workspaces.
- Collections.
- Folders.
- Requests.
- Environments.
- Variables.
- Authentication metadata.
- Execution history.
- Application settings.

### 12.2 Repository interfaces

Services should depend on repository contracts rather than concrete SQL implementations.

For example:

```go
type RequestRepository interface {
    Get(ctx context.Context, id string) (*Request, error)
    Save(ctx context.Context, req *Request) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, workspaceID string) ([]Request, error)
}
```

### 12.3 Database considerations

- Use migrations for schema evolution.
- Enable foreign key constraints.
- Use transactions for multi-entity operations.
- Index frequently queried fields.
- Avoid unnecessary database round trips.
- Handle concurrent access safely.
- Keep large binary response bodies outside the main database when appropriate.

### 12.4 History retention

Execution history should be configurable.

Users should be able to clear history and control retention.

Large response bodies should have separate retention policies from request metadata.

---

## 13. Import and Export

Data migration should be a first-class feature.

The application should support importing existing API client projects and exporting its own workspace data.

Initial formats:

- Postman collections.
- Insomnia exports.
- OpenAPI specifications.
- Native workspace format.

Importers should translate external formats into internal domain models.

They should not directly write arbitrary imported structures into the database.

The import process should:

1. Parse the source.
2. Validate its structure.
3. Map supported entities.
4. Report unsupported features.
5. Detect conflicts.
6. Present a normalized import result.
7. Persist the result transactionally.

Export should preserve relationships, variables, authentication metadata, and collection structure.

Secrets should be excluded or explicitly handled through a secure export policy.

---

## 14. Event and State Management

The application should distinguish between persistent domain state and transient execution state.

Persistent state includes requests, collections, environments, and settings.

Transient state includes:

- Active request executions.
- Loading states.
- Connection states.
- Temporary response data.
- Progress information.
- Runtime errors.

Use an application-level event mechanism to communicate important state changes.

Example events:

```text
request.started
request.completed
request.failed
request.cancelled
collection.updated
environment.changed
workspace.changed
websocket.connected
websocket.disconnected
```

Events should be typed and should not expose internal implementation details.

The GUI should subscribe to relevant application events rather than polling services unnecessarily.

---

## 15. Concurrency and Performance

Go's concurrency model should be used deliberately.

Requirements:

- Independent requests should execute concurrently.
- Each execution should have its own context.
- Response processing should avoid unnecessary copying.
- Large payloads should support streaming.
- Database writes should be coordinated safely.
- Background tasks should have controlled lifecycles.
- Long-running operations should expose cancellation.
- UI updates should be marshalled through the mechanisms supported by MyGO.

Avoid creating unbounded goroutines.

Use bounded concurrency for background operations and bulk execution.

Performance should be measured using realistic workloads rather than assumptions.

---

## 16. Security

Security is a core architectural requirement.

### Credentials

- Never log access tokens, passwords, or API keys.
- Keep secret values separate from ordinary metadata.
- Use secure operating system credential storage where available.
- Avoid including secrets in diagnostic reports.

### TLS

- Verify TLS certificates by default.
- Make insecure certificate verification an explicit, clearly identified option.
- Support custom CA certificates and client certificates.

### Scripting

- Do not expose unrestricted system access to scripts.
- Apply execution limits.
- Restrict access to application capabilities.
- Ensure runtime failures cannot destabilize the main application.

### Local data

- Protect sensitive local data.
- Use secure file permissions.
- Avoid storing plaintext credentials in ordinary configuration files.
- Provide an explicit policy for sensitive data in exports and backups.

---

## 17. Error Handling and Observability

Errors should be structured and categorized.

Suggested categories:

- Validation errors.
- Variable resolution errors.
- Authentication errors.
- DNS errors.
- Connection errors.
- TLS errors.
- Timeout errors.
- HTTP execution errors.
- Response processing errors.
- Storage errors.
- Script execution errors.
- Import/export errors.

Errors should retain their underlying causes where possible.

Use structured logging through `log/slog`.

Logging should support different verbosity levels and must redact sensitive values.

Network errors should be distinguishable from HTTP error responses. An HTTP 4xx or 5xx response is still a successfully completed HTTP exchange.

---

## 18. Testing Strategy

Testing should be part of every subsystem's development.

### Unit tests

Test individual domain models, services, resolvers, authentication providers, and serialization logic.

### Transport tests

Use `httptest` to test:

- Request construction.
- Headers.
- Authentication.
- Redirect behavior.
- Timeout behavior.
- Response parsing.
- Cancellation.
- Streaming.

### Integration tests

Test the interaction between:

- Application services.
- Repository implementations.
- SQLite.
- Transport adapters.
- Variable resolution.
- History persistence.

### Import/export tests

Use representative fixtures from supported formats.

Verify that importing and exporting preserves supported information.

### Concurrency tests

Use the race detector and test concurrent execution, persistence, and cancellation.

Run:

```bash
go test ./...
go test -race ./...
go vet ./...
```

---

## 19. Development Roadmap

Implementation should proceed incrementally.

### Phase 1 — Foundation

- Establish application bootstrap.
- Set up MyGO and Lobe MyGO integration.
- Define domain models.
- Establish service interfaces.
- Implement SQLite storage.
- Introduce migrations.
- Implement workspace management.

### Phase 2 — HTTP Core

- Implement HTTP request models.
- Implement request validation.
- Implement variable resolution.
- Implement HTTP transport.
- Implement response capture.
- Implement cancellation and timeouts.
- Implement execution history.

### Phase 3 — Collections and Environments

- Collection management.
- Nested folders.
- Saved requests.
- Environment management.
- Variable inheritance.
- Collection-level authentication.

### Phase 4 — Authentication

- Basic authentication.
- Bearer authentication.
- API keys.
- Digest authentication.
- OAuth 2.0.
- Secure credential storage.

### Phase 5 — API Testing

- Native assertions.
- Pre-request processing.
- Post-response processing.
- Optional scripting runtime.
- Execution result reporting.

### Phase 6 — Additional Protocols

- GraphQL.
- WebSocket.
- gRPC.

### Phase 7 — Interoperability

- Postman import.
- Insomnia import.
- OpenAPI import.
- Native export.
- Data migration improvements.

### Phase 8 — Optimization and Stability

- Performance profiling.
- Memory optimization.
- Large response handling.
- Concurrency testing.
- Storage optimization.
- Cross-platform validation.

---

## 20. Architectural Constraints

The following constraints must be maintained throughout development:

1. The core application must remain Go-native.
2. MyGO is the GUI framework; Lobe MyGO provides reusable components.
3. Domain logic must not depend on GUI packages.
4. HTTP execution must not depend on GUI state.
5. Storage implementations must be replaceable through interfaces.
6. Protocol implementations must remain isolated.
7. Variable resolution must be centralized.
8. Authentication must be modular.
9. Scripts must not be required for essential functionality.
10. Goja, if retained, must remain an optional and isolated compatibility layer.
11. Sensitive data must never be exposed through ordinary logging.
12. Long-running operations must support cancellation.
13. Large responses must not require unnecessary in-memory buffering.
14. Avoid unnecessary abstractions and premature generalization.
15. Prefer Go's standard library before introducing additional dependencies.

---

## 21. Definition of Done

A feature is considered complete when:

- Its domain behavior is defined.
- Its service implementation is independent of the GUI.
- Errors are handled consistently.
- Relevant unit tests exist.
- Integration tests cover important interactions.
- Context cancellation is respected where applicable.
- Sensitive data is handled appropriately.
- Documentation is updated.
- The feature does not introduce unnecessary architectural coupling.

## 22. Final Architectural Goal

The end result should be a high-performance, native Go API development environment with a modular architecture and a maintainable codebase.

The application should combine the organizational and testing capabilities expected from modern API clients with the efficiency and control of a native Go implementation.

The architecture must make it possible to expand functionality without turning the application into a tightly coupled monolith.

**The GUI is a consumer of the application, not the foundation of the application.**

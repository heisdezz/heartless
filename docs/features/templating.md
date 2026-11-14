# Templating and Dynamic Values

## Purpose

Insert computed or contextual values into requests and configurations at render/send time (variables, generated data, chaining, secrets lookups, prompts).

## Observed Capabilities (Yaak reference)

- **Template engine**: `yaak-templates` crate (parser/renderer, WASM bindings/exports, used by models/rendering and plugins).
- **Template function plugins**: rich built-in function set including `uuid`, `timestamp`, `random`, `hash`, `encode`, `json`, `xml`, `regex`, `fs`, `prompt`, `request`, `response` (chaining), `cookie`, `ctx` (window/context), `1password` integration.
- **Request/response chaining**: extract values from other requests/responses for dependent workflows (`template-function-request`, `template-function-response`).
- **Interactive values**: prompt/user-input functions for runtime-provided values.
- **Rendering context**: templates rendered with environment/workspace/request context before execution; syntax used in URLs, headers, bodies, auth, etc.

## Key Components in Reference

- Crate: `yaak/crates/yaak-templates` (`src`, `pkg`, `bindings`, WASM build artifacts).
- Plugins: `yaak/plugins/template-function-*`.
- Client: `TemplateFunctionDialog.tsx`, editors with template-aware inputs; render utilities in models/client libs.

## Heartless Implementation Notes

- Implement a single interpolation/render pass in the variable-resolution pipeline; avoid ad-hoc template expansion in UI or transport code.
- Support deterministic evaluation order, missing-value policy, and sandboxed function execution (especially FS/process/secret/network-touching functions).
- For chaining (`request`/`response`), make dependencies explicit, prevent cycles, and respect cancellation/timeouts.
- Prefer native/template-function registry in Go; if script-like extensibility is added, keep it optional and isolated from core execution/storage.

# Cookies and Session Data

## Purpose

Manage HTTP cookies across requests and sessions, with user control over sending, storing, and inspecting cookie jars.

## Observed Capabilities (Yaak reference)

- **Cookie models/queries**: cookie jar persistence (`cookie_jars` query module in `yaak-models`, cookie handling in HTTP crate dependencies).
- **Send/store controls**: request/workspace-level settings to send and store cookies (`ResolvedHttpRequestSettings.send_cookies/store_cookies`, timeline setting events).
- **UI management**: cookie dialog/dropdown for viewing/editing cookies (`CookieDialog.tsx`, `CookieDropdown.tsx`).
- **Template access**: `template-function-cookie` plugin for reading/using cookie values in dynamic templates.

## Key Components in Reference

- Crates: `yaak-http` (cookie integration), `yaak-models` (cookie jars/storage).
- Client: cookie dialog/dropdown components.
- Plugins: `template-function-cookie`.

## Heartless Implementation Notes

- Implement cookie jar as domain/storage concern with explicit scope (workspace/environment/request context as defined) and secure persistence.
- Respect per-request/settings toggles during execution; record effective behavior in execution metadata/timeline.
- Provide import/export and redaction policies for sensitive session cookies.
- Ensure template access to cookies is mediated and scoped, not direct unrestricted storage access.

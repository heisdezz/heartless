# Environments and Variables

## Purpose

Manage reusable, scoped values for requests and collections across contexts (dev/staging/prod/local), with secure handling for secrets and deterministic interpolation.

## Observed Capabilities (Yaak reference)

- **Environment entities**: environment models/queries and CRUD/editing flows (`environments` queries, `EnvironmentEditor`, `EnvironmentEditDialog`, `CreateEnvironmentDialog`, `EnvironmentActionsDropdown`).
- **Variables and scoping**: environment/workspace/request-level values and inherited settings patterns (migrations for environments/variables/workspace variables/inherited request settings); public/private environment handling appears in sync logic (`Environment` public flag, private environments treated specially).
- **Variable interpolation**: template system resolves placeholders in request fields (URL, headers, params, auth, bodies, etc.) via `yaak-templates` and render pipelines (`yaak-models` render/path placeholder utilities).
- **UI affordances**: color indicators/pickers, sharable indicators/tooltips, environment selection and editing.

## Key Components in Reference

- Crates: `yaak-models` (environment/variable data), `yaak-templates` (parsing/rendering), sync/watch integration.
- Client: `EnvironmentEditor.tsx`, `EnvironmentEditDialog.tsx`, `EnvironmentActionsDropdown.tsx`, `EnvironmentColorPicker.tsx`, `EnvironmentSharableTooltip.tsx`.

## Heartless Implementation Notes

- Centralize resolution in a dedicated subsystem with explicit precedence (request/folder/collection/environment/workspace/defaults as per design) and circular-reference detection.
- Preserve original saved request definitions; produce resolved request copies for execution/rendering only.
- Mark secret variables, redact in logs/history/UI previews, and control inclusion in exports/sync.
- Keep interpolation syntax consistent (e.g. `{{variable}}`) and provide clear errors for missing/invalid references.

# Filtering and Querying Response Data

## Purpose

Help users locate and extract relevant information from large or structured responses.

## Observed Capabilities (Yaak reference)

- **Filter engine**: `yaak-filter` crate with `fancy-regex`, `sxd-xpath`/`sxd-document`, and JSON processing utilities (tests present).
- **Query languages**: JSONPath and XPath filtering/inspection called out in README ("Filter and inspect responses with JSONPath or XPath").
- **UI integration**: response viewer/filter controls and client dependencies for parsing/formatting (`jsonc-parser`, syntax highlighters, formatting tools) support inspection workflows.

## Key Components in Reference

- Crate: `yaak/crates/yaak-filter`.
- Client: response viewers/panes and editor tooling in `apps/yaak-client/components/responseViewers` and core editors.

## Heartless Implementation Notes

- Provide native, safe query/filter implementations for JSONPath-like and XPath use cases over response bodies.
- Apply filters on-demand/stream-friendly where possible to avoid duplicating large buffers.
- Surface filter errors clearly without mutating stored response data.
- Consider reusable filter service for assertions/tests and response preview.

# Import and Export

## Purpose

Migrate API definitions and workspace data between Yaak/Heartless and external tools/formats, preserving structure and configuration where supported.

## Observed Capabilities (Yaak reference)

- **Importers (plugins)**: `importer-postman`, `importer-postman-environment`, `importer-insomnia`, `importer-openapi`, `importer-curl`, `importer-bruno`, `importer-yaak` (native).
- **Export**: native/export flows and dialogs (`ExportDataDialog.tsx`, `export.rs` in models, `importer-yaak` for Yaak data interchange).
- **Import workflow UI**: `ImportDataDialog.tsx`, `ImportCurl.tsx`, `ImportSourceList.tsx`, import source tracking (`import_sources`, `import_source_resources`, content hash migrations).
- **Mapping concerns**: parse external formats into internal entities (workspaces/collections/folders/requests/environments/auth/variables), detect unsupported/conflicting data, and persist safely.

## Key Components in Reference

- Plugins: `yaak/plugins/importer-*`.
- Client: import/export dialogs and source management components.
- Models: import source/resource tracking, export utilities.

## Heartless Implementation Notes

- Follow design-doc pipeline: parse -> validate -> map -> report unsupported/conflicts -> normalized result -> transactional persist.
- Keep importers/exporters as adapters over domain models/repositories (no direct DB writes from raw external blobs).
- Define explicit secret handling policy for export/import (exclude, redact, or secure-envelope options).
- Provide fixture-based round-trip and compatibility tests per supported format.

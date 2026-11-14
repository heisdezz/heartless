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

### Implemented (P6-IMPORTERS)

Import only (export not yet implemented). Pipeline lives in `internal/services/importer` and runs as:
detect/parse -> validate -> normalize -> map into domain -> collect `Issue`s -> persist draft in one SQLite transaction (`Repository.ImportBatch`).

- **Formats**: Postman collection v2.0/v2.1 (`internal/importer/postman`) and Insomnia v4 (`resources`) / v5 (`collection`) (`internal/importer/insomnia`), selected via the `importer.Importer` registry (`importer.Select`).
- **Imported**: collections, nested folders, requests (method, URL incl. query/path params, headers, raw/urlencoded/formdata/graphql bodies).
- **Reported as issues, not imported (v1 scope)**: environments, variables, auth, scripts, file/binary bodies, and unknown resources (grpc/websocket/api-spec in Insomnia).
- **Conflicts**: duplicate names resolved by `Options.ConflictPolicy` — `rename` (default, appends ` (imported)` / ` (imported N)`) or `skip`.
- **Errors**: `ErrUnsupportedFormat` (no adapter matched), `ErrWorkspaceNotFound`, `ErrEmptyImport`.
- **Persistence**: `requests.headers` / `requests.params` JSON columns via migration v5 `request_params_headers`; all-or-nothing rollback on failure.

### Remaining / follow-ups

- Export flows and secret-handling policy for export.
- OpenAPI, curl, Bruno, and native Yaak importers.
- Environment/variable/auth import (currently reported as unsupported issues).
- Wire `Services.Importers` into `internal/app` bootstrapping (P0-APP-BOOTSTRAP) and add the import UI dialog.

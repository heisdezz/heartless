# GraphQL

## Purpose

Support GraphQL API workflows (queries, mutations, variables, schema-aware editing/introspection) on top of HTTP transport.

## Observed Capabilities (Yaak reference)

- **GraphQL editing UI**: dedicated GraphQL editor/components in client (`components/graphql/*`, `cm6-graphql`, `graphql` dependencies in client).
- **Operations and variables**: author queries/mutations with variables and custom headers (standard for GraphQL-over-HTTP; reflected in request/HTTP tooling).
- **Schema/introspection support**: GraphQL introspection-related model/query artifacts (`graphql_introspections` query module in `yaak-models`).
- **Response inspection**: view GraphQL JSON responses and errors using common response viewers/filtering.

## Key Components in Reference

- Client: `apps/yaak-client/components/graphql/`, CodeMirror GraphQL integration.
- Models/storage: `yaak-models/src/queries/graphql_introspections.rs`.
- Transport: executed via HTTP stack (`yaak-http`) with GraphQL-specific request shaping in app/client layers.

## Heartless Implementation Notes

- Treat GraphQL as an HTTP protocol specialization: domain/request model may carry operation/variables, but transport remains HTTP.
- Provide introspection fetch/cache behavior and structured surfacing of GraphQL `errors` alongside HTTP metadata.
- Reuse variable resolution, auth, headers, TLS/proxy, history, and assertion infrastructure from HTTP.

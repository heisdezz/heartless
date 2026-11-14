# gRPC

## Purpose

Enable unary and streaming gRPC calls with schema/reflection-driven message authoring and response inspection.

## Observed Capabilities (Yaak reference)

- **Protocol adapter**: dedicated `yaak-grpc` crate with `tonic`, `prost`/`prost-reflect`, reflection, transport/TLS integration, and service/method definitions.
- **Schema sources**: reflection and proto/file-based pool loading (`fill_pool_from_reflection`, `fill_pool_from_files`).
- **Request/connection lifecycle**: gRPC request, connection, and event models/queries (`grpc_requests`, `grpc_connections`, `grpc_events`).
- **Streaming awareness**: method metadata tracks client/server streaming flags; manager/connection handling in crate.
- **UI workflows**: proto/service/method selection, request/response panes, connection layout, recent connections (`GrpcRequestPane`, `GrpcResponsePane`, `GrpcProtoSelectionDialog`, `GrpcConnectionLayout`, `GrpcEditor`, `RecentGrpcConnectionsDropdown`).

## Key Components in Reference

- Crate: `yaak/crates/yaak-grpc` (`manager`, `reflection`, `transport`, `codec`, JSON (de)serialization helpers).
- Models: gRPC request/connection/event entities and migrations.
- Client: gRPC-specific layout, panes, editor, and proto selection dialog.

## Heartless Implementation Notes

- Implement gRPC as an isolated transport/protocol adapter; do not leak gRPC-specific types into generic HTTP/domain flows.
- Support reflection and local schema inputs; provide structured method/message validation errors.
- Model streaming events/messages with cancellable contexts and bounded buffering/streaming to UI.
- Plan incremental delivery (unary first, then streaming) consistent with design roadmap.

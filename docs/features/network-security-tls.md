# Network, TLS, Proxy, and DNS

## Purpose

Control how requests reach targets: transport security, client identity, proxy routing, and name resolution overrides.

## Observed Capabilities (Yaak reference)

- **TLS configuration**: `yaak-tls` crate (rustls/native-tls integration points across stack), certificate validation toggles, HTTP version selection, protocol/TLS-related migrations.
- **Client certificates**: client certificate model fields (host/port/cert/key/pfx/passphrase/enabled) and migrations (`client-certificates`).
- **Proxy support**: proxy settings/auth/bypass/disabled fields in models; `yaak-proxy` crate and HTTP transport proxy capabilities (SOCKS/HTTP proxy deps).
- **DNS and resolution controls**: DNS timing capture and DNS overrides models/migrations (`dns-timing`, `dns-overrides`), `DnsOverridesEditor.tsx`.
- **Timeouts/redirects/transport behavior**: request timeout, follow redirects, validate certificates, HTTP version, message size surfaced as resolved settings and timeline events.

## Key Components in Reference

- Crates: `yaak-tls`, `yaak-proxy`, `yaak-http`.
- Models: `ClientCertificate`, `DnsOverride`, proxy settings, resolved HTTP settings, HTTP response events.
- Client: `DnsOverridesEditor.tsx`, settings/request configuration UIs.

## Heartless Implementation Notes

- Default to secure TLS verification; make insecure modes explicit and auditable per request/workspace policy.
- Centralize transport configuration resolution/inheritance and reuse transports/pools safely.
- Support custom CA, mTLS/client certs, proxy auth/bypass, and DNS overrides through transport abstractions.
- Capture connection/TLS/DNS timing metadata when available for diagnostics, without leaking secrets.

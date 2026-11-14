# Authentication

## Purpose

Apply reusable, configurable authentication to outgoing requests with support for common schemes, inheritance, and secure credential handling.

## Observed Capabilities (Yaak reference)

- **Pluggable auth providers (plugins)**: `auth-basic`, `auth-bearer`, `auth-apikey`, `auth-digest`, `auth-oauth1`, `auth-oauth2`, `auth-jwt`, `auth-aws` (SigV4), `auth-ntlm` packages.
- **OAuth 2.0**: multiple grant types (authorization code w/ optional PKCE, client credentials, implicit, resource owner password), token management/refresh, scopes, custom parameters (per plugin README).
- **AWS SigV4**: access key/secret/session token, region/service signing for AWS and S3-compatible APIs (per plugin README).
- **Other schemes**: Basic, Bearer, API key, Digest, JWT, OAuth 1.0, NTLM (plugin set indicates coverage breadth).
- **Configuration surfaces**: request/folder/workspace-level auth configuration via `HttpAuthenticationEditor` and dynamic form-driven plugin inputs.
- **Inheritance and application**: auth metadata stored on requests/entities and applied during request preparation/execution pipeline.

## Key Components in Reference

- Plugins: `yaak/plugins/auth-*` (each with `package.json`, source, some with READMEs/tests/screenshots).
- Client: `HttpAuthenticationEditor.tsx`, `DynamicForm.tsx`.
- Models/storage: request authentication fields/types (`Authentication`, `AuthenticationType` in HTTP models) and secure credential handling integrations.

## Heartless Implementation Notes

- Define a provider interface (apply auth to request, validate config, expose secret fields) and resolve inherited auth before transport execution.
- Isolate OAuth flows/token refresh in dedicated services with secure token persistence; never log credentials/tokens.
- Ship native/core providers for no-auth/basic/bearer/apikey/digest first; add OAuth2/AWS/JWT/NTLM as modular adapters.
- Keep secrets out of normal request metadata exports unless explicitly permitted by policy.

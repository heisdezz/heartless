# Secrets and Encryption

## Purpose

Protect sensitive values (tokens, passwords, keys, private environment data, credentials) at rest, in memory/logs, and during sharing/export.

## Observed Capabilities (Yaak reference)

- **Crypto/keyring integration**: `yaak-crypto` crate with keyring backends (macOS/Windows/Linux secret service), encryption primitives (e.g. chacha20poly1305 deps), master key source abstractions and test support features.
- **Secure storage**: OS keychain/secret-service usage for credentials/tokens; README highlights encrypted secrets and keychain storage.
- **Workspace/request secret handling**: secret-designated variables/fields and workspace encryption settings UI (`WorkspaceEncryptionSetting.tsx`, `EncryptionHelp.tsx`).
- **Private data controls**: private environments excluded from filesystem sync; secret metadata preserved through resolution where applicable.
- **Credential storage for integrations**: Git credentials, auth tokens, template-function secrets (e.g. 1Password plugin) handled via dedicated mechanisms.

## Key Components in Reference

- Crate: `yaak/crates/yaak-crypto` (and keyring usage in plugins/crates).
- Client: encryption settings/help components.
- Sync/models: public/private environment behavior and sensitive-field handling.

## Heartless Implementation Notes

- Enforce "never log secrets" with structured redaction at logger/sink boundaries.
- Separate secret values from ordinary config/entities; store in OS secure storage when available with encrypted fallback policy.
- Define export/backup/sync policies for secrets (default exclude, explicit opt-in secure handling).
- Use secure file permissions for local data and minimize plaintext lifetime in memory where practical.

# Filesystem Sync and Git Integration

## Purpose

Mirror workspace data to the filesystem for portability/version control, and provide Git-aware workflows for collaboration and history outside the app database.

## Observed Capabilities (Yaak reference)

### Filesystem sync
- **Sync engine**: `yaak-sync` crate computes DB vs filesystem candidates and operations (`FsCreate/FsUpdate/FsDelete`, `DbCreate/DbUpdate/DbDelete`, ignore-private handling), tracks sync state/checksums/paths, and watches for changes.
- **Selective sync**: private environments/models can be excluded from filesystem writes (`IgnorePrivate`, public/private environment handling).
- **Workspace integration**: sync directory/settings and migrations related to model changes/sync state; UI setting component `SyncToFilesystemSetting.tsx`.

### Git
- **Git operations crate**: `yaak-git` exposes init/clone, status/branch info, add/unstage/commit/log/diff, fetch/pull/push, merge/checkout/create/delete/rename branches, remotes, restore/reset/stash-adjacent operations (per module exports).
- **Credentials and repo utilities**: credential handling, repository path/ignore checks, worktree/status summaries.
- **UI**: Git-related client components (`components/git/*`), clone dialog (`CloneGitRepositoryDialog.tsx`).

## Key Components in Reference

- Crates: `yaak/crates/yaak-sync`, `yaak/crates/yaak-git`.
- Models: sync states, model change tracking (`sync_states`, `model_changes`).
- Client: sync settings and Git dialogs/components.

## Heartless Implementation Notes

- Keep sync as an explicit service with deterministic diff/apply semantics and conflict reporting; never couple core request execution to Git.
- Define canonical on-disk format (stable ordering, secret exclusion/redaction) suitable for diffs/reviews.
- Run filesystem and Git operations off the UI thread with progress/cancellation and structured errors.
- Treat Git integration as optional/layered; core data portability should work via export/sync even without Git.

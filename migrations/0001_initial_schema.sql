-- Initial schema for Heartless (applied by internal/storage/sqlite migrator)

CREATE TABLE IF NOT EXISTS schema_migrations(
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspaces(
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS environments(
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_environments_workspace_id ON environments (
  workspace_id
);

CREATE TABLE IF NOT EXISTS variables(
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  value TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  secret INTEGER NOT NULL DEFAULT 0,
  scope TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_variables_workspace_id ON variables (
  workspace_id
);

CREATE INDEX IF NOT EXISTS idx_variables_environment_id ON variables (
  environment_id
);

CREATE TABLE IF NOT EXISTS workspace_state(
  workspace_id TEXT PRIMARY KEY,
  active_environment_id TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS execution_history(
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  method TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  status_code INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  status_text TEXT NOT NULL DEFAULT '',
  response_headers TEXT NOT NULL DEFAULT '{}',
  body TEXT NOT NULL DEFAULT '',
  content_type TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  response_id TEXT NOT NULL DEFAULT '',
  response_request_id TEXT NOT NULL DEFAULT '',
  response_duration_ns INTEGER NOT NULL DEFAULT 0,
  effective_url TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_execution_history_workspace_created ON execution_history (
  workspace_id,
  created_at DESC
);

CREATE INDEX IF NOT EXISTS idx_execution_history_request_id ON execution_history (
  request_id
);

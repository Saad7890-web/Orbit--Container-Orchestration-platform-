PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS stacks (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    labels_json TEXT NOT NULL DEFAULT '{}',
    config_hash TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS services (
    stack_name TEXT NOT NULL,
    name TEXT NOT NULL,
    spec_json TEXT NOT NULL,
    config_hash TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'unknown',
    health_status TEXT NOT NULL DEFAULT 'unknown',
    last_error TEXT NOT NULL DEFAULT '',
    last_seen_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (stack_name, name),
    FOREIGN KEY (stack_name) REFERENCES stacks(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS jobs (
    stack_name TEXT NOT NULL,
    name TEXT NOT NULL,
    spec_json TEXT NOT NULL,
    config_hash TEXT NOT NULL DEFAULT '',
    last_status TEXT NOT NULL DEFAULT 'pending',
    last_error TEXT NOT NULL DEFAULT '',
    last_run_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (stack_name, name),
    FOREIGN KEY (stack_name) REFERENCES stacks(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS triggers (
    stack_name TEXT NOT NULL,
    name TEXT NOT NULL,
    spec_json TEXT NOT NULL,
    config_hash TEXT NOT NULL DEFAULT '',
    last_status TEXT NOT NULL DEFAULT 'pending',
    last_error TEXT NOT NULL DEFAULT '',
    last_fired_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (stack_name, name),
    FOREIGN KEY (stack_name) REFERENCES stacks(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    workload_kind TEXT NOT NULL,
    workload_name TEXT NOT NULL,
    trigger_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    exit_code INTEGER,
    error TEXT NOT NULL DEFAULT '',
    logs_ref TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS runtime_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_services_stack_name ON services(stack_name);
CREATE INDEX IF NOT EXISTS idx_jobs_stack_name ON jobs(stack_name);
CREATE INDEX IF NOT EXISTS idx_triggers_stack_name ON triggers(stack_name);
CREATE INDEX IF NOT EXISTS idx_executions_workload ON executions(workload_kind, workload_name);
CREATE INDEX IF NOT EXISTS idx_executions_started_at ON executions(started_at DESC);
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL
);

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    root_hash TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);

CREATE TABLE clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    UNIQUE(name, version)
);

CREATE TABLE models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    display_name TEXT NOT NULL UNIQUE
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    client_id INTEGER NOT NULL REFERENCES clients(id),
    model_id INTEGER NOT NULL REFERENCES models(id),
    started_at TEXT NOT NULL,
    ended_at TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    title TEXT NOT NULL DEFAULT ''
);

CREATE TABLE events (
    id TEXT PRIMARY KEY,
    schema_version INTEGER NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    client_id INTEGER NOT NULL REFERENCES clients(id),
    model_id INTEGER NOT NULL REFERENCES models(id),
    occurred_at TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    provenance TEXT NOT NULL,
    precision TEXT NOT NULL CHECK (precision IN ('exact', 'estimated', 'unavailable'))
);

CREATE TABLE git_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    head_sha TEXT NOT NULL DEFAULT '',
    branch TEXT NOT NULL DEFAULT '',
    dirty INTEGER NOT NULL DEFAULT 0,
    captured_at TEXT NOT NULL
);

CREATE TABLE validations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    command_label TEXT NOT NULL,
    status TEXT NOT NULL,
    duration_ms INTEGER,
    summary TEXT NOT NULL DEFAULT '',
    occurred_at TEXT NOT NULL
);

CREATE TABLE usage_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cached_tokens INTEGER,
    provenance TEXT NOT NULL,
    precision TEXT NOT NULL,
    occurred_at TEXT NOT NULL
);

CREATE TABLE cost_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    amount_micros INTEGER,
    currency TEXT NOT NULL,
    catalog_version TEXT,
    exchange_rate_version TEXT,
    provenance TEXT NOT NULL,
    precision TEXT NOT NULL,
    occurred_at TEXT NOT NULL
);

CREATE TABLE quota_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    quota_name TEXT NOT NULL,
    remaining_micros INTEGER,
    reset_at TEXT,
    provenance TEXT NOT NULL,
    precision TEXT NOT NULL,
    captured_at TEXT NOT NULL
);

CREATE TABLE memories (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    content TEXT NOT NULL,
    confidence REAL NOT NULL,
    source_session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    supersedes_id TEXT REFERENCES memories(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE context_capsules (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    token_estimate INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE TABLE diagnoses (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    severity TEXT NOT NULL,
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    explanation TEXT NOT NULL,
    evidence_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE comparisons (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    left_session_id TEXT NOT NULL REFERENCES sessions(id),
    right_session_id TEXT NOT NULL REFERENCES sessions(id),
    result_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE replays (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    manifest_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE consents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope TEXT NOT NULL,
    granted INTEGER NOT NULL,
    policy_version TEXT NOT NULL,
    recorded_at TEXT NOT NULL
);

CREATE TABLE price_catalog_versions (
    version TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    effective_at TEXT NOT NULL,
    imported_at TEXT NOT NULL
);

CREATE TABLE exchange_rate_versions (
    version TEXT PRIMARY KEY,
    base_currency TEXT NOT NULL,
    quote_currency TEXT NOT NULL,
    rate TEXT NOT NULL,
    source TEXT NOT NULL,
    effective_at TEXT NOT NULL,
    imported_at TEXT NOT NULL
);

CREATE INDEX idx_sessions_project_started ON sessions(project_id, started_at DESC);
CREATE INDEX idx_events_session_time ON events(session_id, occurred_at, id);
CREATE INDEX idx_events_project_time ON events(project_id, occurred_at DESC);
CREATE INDEX idx_events_type_time ON events(event_type, occurred_at DESC);
CREATE INDEX idx_validations_session_time ON validations(session_id, occurred_at DESC);
CREATE INDEX idx_usage_session_time ON usage_records(session_id, occurred_at DESC);
CREATE INDEX idx_cost_session_time ON cost_records(session_id, occurred_at DESC);
CREATE INDEX idx_quota_time ON quota_snapshots(captured_at DESC);
CREATE INDEX idx_memories_project_kind ON memories(project_id, kind, updated_at DESC);
CREATE INDEX idx_diagnoses_session_severity ON diagnoses(session_id, severity, created_at DESC);

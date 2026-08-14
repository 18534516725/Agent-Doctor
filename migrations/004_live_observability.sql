CREATE TABLE model_requests (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    client_id INTEGER NOT NULL REFERENCES clients(id),
    model_id INTEGER NOT NULL REFERENCES models(id),
    protocol TEXT NOT NULL,
    method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    status_code INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    first_byte_ms INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cached_tokens INTEGER,
    reasoning_tokens INTEGER,
    usage_precision TEXT NOT NULL CHECK (usage_precision IN ('exact', 'estimated', 'unavailable')),
    usage_provenance TEXT NOT NULL,
    cost_amount_micros INTEGER,
    cost_currency TEXT NOT NULL DEFAULT 'USD',
    cost_precision TEXT NOT NULL CHECK (cost_precision IN ('exact', 'estimated', 'unavailable')),
    cost_provenance TEXT NOT NULL
);

CREATE TABLE conversation_messages (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES model_requests(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT NOT NULL,
    tool_name TEXT NOT NULL DEFAULT '',
    tool_payload_json TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE(request_id, sequence)
);

CREATE TABLE client_connections (
    client_key TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    detected INTEGER NOT NULL CHECK (detected IN (0, 1)),
    state TEXT NOT NULL CHECK (state IN ('unavailable', 'detected', 'connected', 'active', 'error')),
    capability TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    last_heartbeat_at TEXT,
    updated_at TEXT NOT NULL
);

CREATE TABLE analysis_snapshots (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    provenance TEXT NOT NULL,
    precision TEXT NOT NULL CHECK (precision IN ('exact', 'estimated', 'unavailable')),
    created_at TEXT NOT NULL
);

CREATE INDEX idx_model_requests_session_time ON model_requests(session_id, started_at DESC);
CREATE INDEX idx_model_requests_project_time ON model_requests(project_id, started_at DESC);
CREATE INDEX idx_conversation_messages_session_time ON conversation_messages(session_id, created_at, sequence);
CREATE INDEX idx_client_connections_updated ON client_connections(updated_at DESC);
CREATE INDEX idx_analysis_snapshots_session_time ON analysis_snapshots(session_id, created_at DESC);

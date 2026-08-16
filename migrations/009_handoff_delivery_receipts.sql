CREATE TABLE handoff_delivery_receipts (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    source_client TEXT NOT NULL,
    target_client TEXT NOT NULL,
    memory_count INTEGER NOT NULL DEFAULT 0 CHECK (memory_count >= 0),
    delivered_at TEXT NOT NULL
);

CREATE INDEX idx_handoff_delivery_project_time
    ON handoff_delivery_receipts(project_id, delivered_at DESC);

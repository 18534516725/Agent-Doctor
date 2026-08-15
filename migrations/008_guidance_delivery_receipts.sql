CREATE TABLE guidance_delivery_receipts (
    session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    client TEXT NOT NULL,
    decision_id TEXT NOT NULL,
    decision_kind TEXT NOT NULL CHECK (decision_kind IN ('continue','advise','redirect','ask','block','verify')),
    control_level TEXT NOT NULL CHECK (control_level IN ('observe','guide','guard','autopilot')),
    delivery_count INTEGER NOT NULL DEFAULT 1 CHECK (delivery_count > 0),
    first_delivered_at TEXT NOT NULL,
    delivered_at TEXT NOT NULL
);

CREATE INDEX idx_guidance_delivery_project_time
    ON guidance_delivery_receipts(project_id, delivered_at DESC);

CREATE TABLE guidance_decisions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('continue','advise','redirect','ask','block','verify')),
    severity TEXT NOT NULL CHECK (severity IN ('info','warning','high','critical')),
    payload_json TEXT NOT NULL,
    evidence_fingerprint TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(session_id, evidence_fingerprint)
);

CREATE TABLE project_guidance_settings (
    project_id TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    control_level TEXT NOT NULL DEFAULT 'guide' CHECK (control_level IN ('observe','guide','guard','autopilot')),
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_guidance_session_created ON guidance_decisions(session_id, created_at DESC);
CREATE INDEX idx_guidance_project_created ON guidance_decisions(project_id, created_at DESC);

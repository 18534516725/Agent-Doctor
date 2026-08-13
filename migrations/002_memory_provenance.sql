ALTER TABLE memories ADD COLUMN source TEXT NOT NULL DEFAULT 'inferred';
ALTER TABLE memories ADD COLUMN observation_count INTEGER NOT NULL DEFAULT 1;
ALTER TABLE memories ADD COLUMN state TEXT NOT NULL DEFAULT 'candidate'
    CHECK (state IN ('candidate', 'active', 'disabled', 'deleted'));
ALTER TABLE memories ADD COLUMN expires_at TEXT;
ALTER TABLE memories ADD COLUMN deleted_at TEXT;

CREATE INDEX idx_memories_project_state_source
    ON memories(project_id, state, source, confidence DESC, updated_at DESC);

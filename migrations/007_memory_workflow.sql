ALTER TABLE memories ADD COLUMN source_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_memories_project_updated ON memories(project_id, updated_at DESC);

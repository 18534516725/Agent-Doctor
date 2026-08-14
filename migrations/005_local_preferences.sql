CREATE TABLE privacy_settings (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    capture_prompts INTEGER NOT NULL CHECK (capture_prompts IN (0, 1)),
    capture_file_contents INTEGER NOT NULL CHECK (capture_file_contents IN (0, 1)),
    retention_days INTEGER NOT NULL CHECK (retention_days BETWEEN 1 AND 3650),
    updated_at TEXT NOT NULL
);

INSERT INTO privacy_settings(singleton, capture_prompts, capture_file_contents, retention_days, updated_at)
VALUES(1, 1, 0, 30, '1970-01-01T00:00:00Z');

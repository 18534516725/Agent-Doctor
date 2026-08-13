ALTER TABLE cost_records ADD COLUMN amount_precision TEXT NOT NULL DEFAULT 'unavailable'
    CHECK (amount_precision IN ('exact', 'estimated', 'unavailable'));
ALTER TABLE cost_records ADD COLUMN amount_provenance TEXT NOT NULL DEFAULT 'cost-not-available';
ALTER TABLE cost_records ADD COLUMN price_version TEXT NOT NULL DEFAULT '';

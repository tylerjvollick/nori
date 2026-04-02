-- Add station_id and linked_sop_template_id columns to sop_step
-- Note: station_id has no FK constraint yet because the station table
-- does not exist. The FK will be added in the Station migration (Phase 2).

ALTER TABLE sop_step
    ADD COLUMN station_id UUID;

ALTER TABLE sop_step
    ADD COLUMN linked_sop_template_id INT;

-- FK for linked_sop_template_id → sop_template (exists now)
ALTER TABLE sop_step
    ADD CONSTRAINT fk_sop_step_linked_sop_template
    FOREIGN KEY (linked_sop_template_id)
    REFERENCES sop_template (id)
    ON DELETE SET NULL;

-- Index on station_id for future station-scoped queries
CREATE INDEX IF NOT EXISTS idx_sop_step_station_id
    ON sop_step (station_id);

-- Index on linked_sop_template_id for lookups
CREATE INDEX IF NOT EXISTS idx_sop_step_linked_sop_template_id
    ON sop_step (linked_sop_template_id);

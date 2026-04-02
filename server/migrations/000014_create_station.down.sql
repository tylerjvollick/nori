-- Remove FK constraint on sop_step.station_id (added in this migration)
ALTER TABLE sop_step
    DROP CONSTRAINT IF EXISTS fk_sop_step_station;

-- Remove indexes
DROP INDEX IF EXISTS idx_station_space_id_display_order;
DROP INDEX IF EXISTS idx_station_space_id;

-- Drop station table
DROP TABLE IF EXISTS station;

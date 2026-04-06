-- Remove indexes
DROP INDEX IF EXISTS idx_station_space_id_display_order;
DROP INDEX IF EXISTS idx_station_space_id;

-- Drop station table
DROP TABLE IF EXISTS station;

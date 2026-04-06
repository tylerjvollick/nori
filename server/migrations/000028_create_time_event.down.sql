-- Drop indexes
DROP INDEX IF EXISTS idx_time_event_user_timestamp;
DROP INDEX IF EXISTS idx_time_event_space_timestamp;
DROP INDEX IF EXISTS idx_time_event_station;
DROP INDEX IF EXISTS idx_time_event_task;
DROP INDEX IF EXISTS idx_time_event_user;
DROP INDEX IF EXISTS idx_time_event_space;

-- Drop table
DROP TABLE IF EXISTS time_event;

-- Drop enum types
DROP TYPE IF EXISTS time_event_source;
DROP TYPE IF EXISTS time_event_type;

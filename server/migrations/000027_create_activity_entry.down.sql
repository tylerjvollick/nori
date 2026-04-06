-- Drop indexes
DROP INDEX IF EXISTS idx_activity_entry_type;
DROP INDEX IF EXISTS idx_activity_entry_user;
DROP INDEX IF EXISTS idx_activity_entry_step_task;
DROP INDEX IF EXISTS idx_activity_entry_task_chronological;
DROP INDEX IF EXISTS idx_activity_entry_task;

-- Drop table
DROP TABLE IF EXISTS activity_entry;

-- Drop enum type
DROP TYPE IF EXISTS activity_entry_type;

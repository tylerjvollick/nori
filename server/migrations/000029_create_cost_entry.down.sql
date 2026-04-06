-- Drop indexes
DROP INDEX IF EXISTS idx_cost_entry_created_by;
DROP INDEX IF EXISTS idx_cost_entry_time_event;
DROP INDEX IF EXISTS idx_cost_entry_material;
DROP INDEX IF EXISTS idx_cost_entry_task_cost_type;
DROP INDEX IF EXISTS idx_cost_entry_task;

-- Drop table
DROP TABLE IF EXISTS cost_entry;

-- Drop enum type
DROP TYPE IF EXISTS cost_type;

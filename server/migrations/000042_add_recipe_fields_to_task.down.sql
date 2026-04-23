-- Remove batch_size and estimated_time_secs columns from task table
ALTER TABLE task DROP COLUMN IF EXISTS estimated_time_secs;
ALTER TABLE task DROP COLUMN IF EXISTS batch_size;

-- Note: PostgreSQL does not support removing enum values.
-- The 'recipe' value will remain in task_type but is harmless.

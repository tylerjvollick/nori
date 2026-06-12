-- Reverse: restore active/paused statuses, restore duration_secs on time_entry

-- 1. Restore duration_secs from elapsed_secs
ALTER TABLE time_entry ADD COLUMN duration_secs INT;
UPDATE time_entry SET duration_secs = elapsed_secs;

-- 2. Drop session fields from time_entry
ALTER TABLE time_entry DROP COLUMN elapsed_secs;
ALTER TABLE time_entry DROP COLUMN is_paused;
ALTER TABLE time_entry DROP COLUMN paused_at;
ALTER TABLE time_entry DROP COLUMN resumed_at;

-- 3. Restore paused_at on task
ALTER TABLE task ADD COLUMN paused_at TIMESTAMPTZ;

-- 4. Restore old ENUM with active/paused
ALTER TYPE task_status RENAME TO task_status_new;
CREATE TYPE task_status AS ENUM ('open', 'active', 'paused', 'done', 'skipped', 'cancelled');
ALTER TABLE task
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE task_status
        USING CASE
            WHEN status::text = 'in_progress' THEN 'active'::task_status
            ELSE status::text::task_status
        END,
    ALTER COLUMN status SET DEFAULT 'open';
DROP TYPE task_status_new;

-- Decouple task status from timer state.
-- Replace active/paused with in_progress. Timer state moves to time_entry table.

-- 1. Migrate task_status ENUM: add in_progress, remove active/paused
ALTER TYPE task_status RENAME TO task_status_old;
CREATE TYPE task_status AS ENUM ('open', 'in_progress', 'done', 'skipped', 'cancelled');
ALTER TABLE task
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE task_status
        USING CASE
            WHEN status::text IN ('active', 'paused') THEN 'in_progress'::task_status
            ELSE status::text::task_status
        END,
    ALTER COLUMN status SET DEFAULT 'open';
DROP TYPE task_status_old;

-- 2. Drop paused_at from task (timer state is on time_entry now)
ALTER TABLE task DROP COLUMN IF EXISTS paused_at;

-- 3. Add session fields to time_entry
ALTER TABLE time_entry ADD COLUMN elapsed_secs INT NOT NULL DEFAULT 0;
ALTER TABLE time_entry ADD COLUMN is_paused BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE time_entry ADD COLUMN paused_at TIMESTAMPTZ;
ALTER TABLE time_entry ADD COLUMN resumed_at TIMESTAMPTZ;

-- 4. Migrate existing duration_secs data into elapsed_secs, then drop
UPDATE time_entry SET elapsed_secs = COALESCE(duration_secs, 0);
ALTER TABLE time_entry DROP COLUMN duration_secs;

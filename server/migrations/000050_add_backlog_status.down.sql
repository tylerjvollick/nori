-- Remove 'backlog' from task_status. PostgreSQL cannot drop a single enum
-- value, so rebuild the type (same approach as migration 000048). Any backlog
-- rows are remapped to 'open'.
ALTER TYPE task_status RENAME TO task_status_old;
CREATE TYPE task_status AS ENUM ('open', 'in_progress', 'done', 'skipped', 'cancelled');
ALTER TABLE task
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE task_status
        USING CASE
            WHEN status::text = 'backlog' THEN 'open'::task_status
            ELSE status::text::task_status
        END,
    ALTER COLUMN status SET DEFAULT 'open';
DROP TYPE task_status_old;

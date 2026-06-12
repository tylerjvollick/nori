-- Add 'backlog' to task_status for jobs that are confirmed (e.g. quote
-- accepted) but not yet scheduled. Flow: backlog -> open -> in_progress -> done.
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'backlog' BEFORE 'open';

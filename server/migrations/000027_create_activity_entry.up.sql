-- Create activity_entry table for chronological task activity logging.
-- Records step transitions, interruptions, comments, status changes, and more.
-- FK to task is added in 000038 after the task table exists.

CREATE TYPE activity_entry_type AS ENUM (
    'status_change',
    'step_started',
    'step_completed',
    'step_paused',
    'step_resumed',
    'step_skipped',
    'comment',
    'interruption',
    'assignment_change',
    'link_added',
    'recipe_edited',
    'cost_logged',
    'task_created'
);

CREATE TABLE activity_entry (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    entry_type activity_entry_type NOT NULL,
    description TEXT NOT NULL,
    linked_task_id VARCHAR(255),
    step_task_id VARCHAR(255),
    duration_seconds INT,
    old_value TEXT,
    new_value TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX idx_activity_entry_task ON activity_entry(task_id);
CREATE INDEX idx_activity_entry_task_chronological ON activity_entry(task_id, created_at ASC);
CREATE INDEX idx_activity_entry_step_task ON activity_entry(step_task_id) WHERE step_task_id IS NOT NULL;
CREATE INDEX idx_activity_entry_user ON activity_entry(user_id);
CREATE INDEX idx_activity_entry_type ON activity_entry(entry_type);

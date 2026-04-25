-- Add recipe formula fields to task table
ALTER TABLE task ADD COLUMN estimated_time_formula TEXT;
ALTER TABLE task ADD COLUMN estimated_time_from_recipe_secs INT;

-- Create time_entry table for time tracking
CREATE TABLE time_entry (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id       VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    space_id      UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    logged_by_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL,
    ended_at      TIMESTAMPTZ,
    duration_secs INT,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_time_entry_task_id ON time_entry(task_id);
CREATE INDEX idx_time_entry_space_id ON time_entry(space_id);
CREATE INDEX idx_time_entry_logged_by_id ON time_entry(logged_by_id);

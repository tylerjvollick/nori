-- Create dep_type enum
DO $$ BEGIN
    CREATE TYPE dep_type AS ENUM ('blocks', 'waits_for', 'related');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Create task_dep table
CREATE TABLE task_dep (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_task_id VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    to_task_id VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    type dep_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Prevent duplicate edges between the same pair of tasks
    CONSTRAINT uq_task_dep_from_to UNIQUE (from_task_id, to_task_id),

    -- Prevent self-referential dependencies
    CONSTRAINT chk_task_dep_no_self CHECK (from_task_id <> to_task_id)
);

-- Index on from_task_id for GetBlockers queries (from_task_id = the blocked task)
CREATE INDEX idx_task_dep_from_task_id ON task_dep(from_task_id);

-- Index on to_task_id for GetDependents queries (to_task_id = the blocker/upstream task)
CREATE INDEX idx_task_dep_to_task_id ON task_dep(to_task_id);

-- Index on type for filtering by dependency kind
CREATE INDEX idx_task_dep_type ON task_dep(type);

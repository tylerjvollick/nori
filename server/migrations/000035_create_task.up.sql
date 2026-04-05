-- Create task_type enum
DO $$ BEGIN
    CREATE TYPE task_type AS ENUM ('job', 'task', 'milestone', 'gate');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Create task_status enum
DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('open', 'active', 'paused', 'done', 'skipped', 'cancelled');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Create task table
CREATE TABLE task (
    id VARCHAR(255) PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id VARCHAR(255) REFERENCES task(id) ON DELETE CASCADE,
    recipe_id UUID,
    recipe_version_id INT,
    station_id UUID REFERENCES station(id) ON DELETE SET NULL,
    customer_id UUID REFERENCES customer(id) ON DELETE SET NULL,
    assigned_to_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    type task_type NOT NULL DEFAULT 'task',
    status task_status NOT NULL DEFAULT 'open',
    title VARCHAR(500) NOT NULL,
    description TEXT,
    priority INT NOT NULL DEFAULT 0,
    display_order INT NOT NULL DEFAULT 0,
    due_date TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    actual_time_secs INT NOT NULL DEFAULT 0,
    deviation_notes TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on space_id for space-scoped queries
CREATE INDEX idx_task_space_id ON task(space_id);

-- Index on parent_id for fetching children
CREATE INDEX idx_task_parent_id ON task(parent_id);

-- Index on status for filtering by status
CREATE INDEX idx_task_status ON task(status);

-- Index on space_id + status for common filtered listing
CREATE INDEX idx_task_space_id_status ON task(space_id, status);

-- Index on space_id + type for filtering jobs vs tasks
CREATE INDEX idx_task_space_id_type ON task(space_id, type);

-- Index on station_id for station-scoped views
CREATE INDEX idx_task_station_id ON task(station_id);

-- Index on assigned_to_id for user workload queries
CREATE INDEX idx_task_assigned_to_id ON task(assigned_to_id);

-- Index on customer_id for customer-scoped queries
CREATE INDEX idx_task_customer_id ON task(customer_id);

-- Index on parent_id + display_order for ordered sibling listing
CREATE INDEX idx_task_parent_id_display_order ON task(parent_id, display_order);

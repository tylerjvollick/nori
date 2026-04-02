-- Create station table
CREATE TABLE station (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    display_order INT NOT NULL DEFAULT 0,
    wip_limit INT NOT NULL DEFAULT 1,
    buffer_size INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on space_id for space-scoped queries
CREATE INDEX idx_station_space_id ON station(space_id);

-- Index on space_id + display_order for ordered listing
CREATE INDEX idx_station_space_id_display_order ON station(space_id, display_order);

-- Add FK constraint on sop_step.station_id → station.id (deferred from task 1.3)
ALTER TABLE sop_step
    ADD CONSTRAINT fk_sop_step_station
    FOREIGN KEY (station_id)
    REFERENCES station(id)
    ON DELETE SET NULL;

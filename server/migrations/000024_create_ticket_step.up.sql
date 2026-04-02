-- Create ticket_step table for live execution steps within tickets.
-- When a ticket is created with a linked SOP, the SOP's steps are copied
-- as ticket steps. Steps can also be added ad-hoc during first-time capture.

CREATE TABLE ticket_step (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES ticket(id) ON DELETE CASCADE,
    sop_step_id INT REFERENCES sop_step(id) ON DELETE SET NULL,
    station_id UUID REFERENCES station(id) ON DELETE SET NULL,
    display_order INT NOT NULL DEFAULT 0,
    title VARCHAR(500) NOT NULL,
    instructions TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    assigned_to_id UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    actual_time_seconds INT NOT NULL DEFAULT 0,
    deviation_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_ticket_step_status
        CHECK (status IN ('pending', 'in_progress', 'paused', 'completed', 'skipped'))
);

-- Indexes
CREATE INDEX idx_ticket_step_ticket_id ON ticket_step(ticket_id);
CREATE INDEX idx_ticket_step_station_id ON ticket_step(station_id) WHERE station_id IS NOT NULL;
CREATE INDEX idx_ticket_step_assigned_to_id ON ticket_step(assigned_to_id) WHERE assigned_to_id IS NOT NULL;
CREATE INDEX idx_ticket_step_status ON ticket_step(ticket_id, status);
CREATE INDEX idx_ticket_step_order ON ticket_step(ticket_id, display_order);

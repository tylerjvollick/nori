-- Create ticket_sub_step table for checklist-style detail steps within ticket steps.
-- Not individually timed. Copied from SOP sub-steps when the ticket is created,
-- or added ad-hoc during first-time capture.

CREATE TABLE ticket_sub_step (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    ticket_step_id UUID NOT NULL REFERENCES ticket_step(id) ON DELETE CASCADE,
    sop_sub_step_id INT REFERENCES sop_sub_step(id) ON DELETE SET NULL,
    display_order INT NOT NULL DEFAULT 0,
    title VARCHAR(500) NOT NULL,
    instructions TEXT,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_ticket_sub_step_ticket_step_id ON ticket_sub_step(ticket_step_id);
CREATE INDEX idx_ticket_sub_step_order ON ticket_sub_step(ticket_step_id, display_order);

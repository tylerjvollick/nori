-- Create ticket table
CREATE TABLE ticket (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    ticket_type_id UUID NOT NULL REFERENCES ticket_type(id),
    parent_ticket_id UUID REFERENCES ticket(id) ON DELETE SET NULL,
    status_id UUID NOT NULL REFERENCES status_definition(id),
    sop_template_id INT REFERENCES sop_template(id) ON DELETE SET NULL,
    sop_version_id INT REFERENCES sop_version(id) ON DELETE SET NULL,
    customer_id UUID REFERENCES customer(id) ON DELETE SET NULL,
    assigned_to_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    ticket_number VARCHAR(50) NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    due_date TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE UNIQUE INDEX idx_ticket_number ON ticket(ticket_number);
CREATE INDEX idx_ticket_space_id ON ticket(space_id);
CREATE INDEX idx_ticket_ticket_type_id ON ticket(ticket_type_id);
CREATE INDEX idx_ticket_status_id ON ticket(status_id);
CREATE INDEX idx_ticket_parent_ticket_id ON ticket(parent_ticket_id);
CREATE INDEX idx_ticket_customer_id ON ticket(customer_id);
CREATE INDEX idx_ticket_assigned_to_id ON ticket(assigned_to_id);
CREATE INDEX idx_ticket_created_by_id ON ticket(created_by_id);
CREATE INDEX idx_ticket_priority ON ticket(space_id, priority);
CREATE INDEX idx_ticket_due_date ON ticket(space_id, due_date) WHERE due_date IS NOT NULL;

-- Parent/child constraint: a ticket cannot be a parent if it already has a parent.
-- This enforces one level of nesting only. Implemented as a trigger since
-- a check constraint cannot reference other rows.
CREATE OR REPLACE FUNCTION check_ticket_nesting_depth() RETURNS TRIGGER AS $$
BEGIN
    -- If this ticket has a parent, ensure the parent does not itself have a parent
    IF NEW.parent_ticket_id IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM ticket
            WHERE id = NEW.parent_ticket_id
              AND parent_ticket_id IS NOT NULL
        ) THEN
            RAISE EXCEPTION 'Cannot nest tickets more than one level deep: parent ticket already has a parent';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_ticket_nesting_depth
    BEFORE INSERT OR UPDATE OF parent_ticket_id ON ticket
    FOR EACH ROW
    EXECUTE FUNCTION check_ticket_nesting_depth();

-- Add FK constraint on ticket_tag (deferred from migration 000017)
ALTER TABLE ticket_tag
    ADD CONSTRAINT fk_ticket_tag_ticket_id
    FOREIGN KEY (ticket_id) REFERENCES ticket(id) ON DELETE CASCADE;

-- Sequence table for auto-generating ticket numbers per ticket type
CREATE TABLE ticket_number_sequence (
    ticket_type_id UUID NOT NULL REFERENCES ticket_type(id) ON DELETE CASCADE,
    last_number INT NOT NULL DEFAULT 0,
    PRIMARY KEY (ticket_type_id)
);

-- Create ticket_link table for cross-ticket relationships, including cross-space.
-- Parent/child relationships use Ticket.ParentTicketID instead.
-- TicketLink is for peer relationships: blocks, relates_to, created_from.

CREATE TYPE link_type AS ENUM ('created_from', 'blocks', 'blocked_by', 'relates_to');

CREATE TABLE ticket_link (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    source_ticket_id UUID NOT NULL REFERENCES ticket(id) ON DELETE CASCADE,
    target_ticket_id UUID NOT NULL REFERENCES ticket(id) ON DELETE CASCADE,
    link_type link_type NOT NULL,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Prevent duplicate links between the same pair with the same type
    CONSTRAINT uq_ticket_link_source_target_type UNIQUE (source_ticket_id, target_ticket_id, link_type),

    -- Prevent self-links
    CONSTRAINT chk_ticket_link_no_self CHECK (source_ticket_id != target_ticket_id)
);

-- Indexes for querying links in both directions
CREATE INDEX idx_ticket_link_source ON ticket_link(source_ticket_id);
CREATE INDEX idx_ticket_link_target ON ticket_link(target_ticket_id);
CREATE INDEX idx_ticket_link_type ON ticket_link(link_type);

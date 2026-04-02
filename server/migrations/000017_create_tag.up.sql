-- Create tag table for reusable labels
CREATE TABLE tag (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique constraint: tag name must be unique within a space
CREATE UNIQUE INDEX idx_tag_space_id_name ON tag(space_id, name);

-- Index on space_id for space-scoped queries
CREATE INDEX idx_tag_space_id ON tag(space_id);

-- Join table: SOPTemplate <-> Tag
CREATE TABLE sop_template_tag (
    sop_template_id INT NOT NULL REFERENCES sop_template(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (sop_template_id, tag_id)
);

-- Indexes for querying tags by template and templates by tag
CREATE INDEX idx_sop_template_tag_template_id ON sop_template_tag(sop_template_id);
CREATE INDEX idx_sop_template_tag_tag_id ON sop_template_tag(tag_id);

-- Join table: Ticket <-> Tag
-- Note: ticket table does not exist yet. The ticket_id column has no FK constraint.
-- The FK constraint will be added in Phase 3 when the Ticket model is created.
CREATE TABLE ticket_tag (
    ticket_id UUID NOT NULL,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (ticket_id, tag_id)
);

-- Index for querying tags by ticket and tickets by tag
CREATE INDEX idx_ticket_tag_ticket_id ON ticket_tag(ticket_id);
CREATE INDEX idx_ticket_tag_tag_id ON ticket_tag(tag_id);

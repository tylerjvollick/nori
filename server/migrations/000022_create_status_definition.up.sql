-- Create status_category enum type
CREATE TYPE status_category AS ENUM ('todo', 'in_progress', 'done');

-- Create status_definition table
CREATE TABLE status_definition (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    ticket_type_id UUID NOT NULL REFERENCES ticket_type(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    category status_category NOT NULL DEFAULT 'todo',
    color VARCHAR(7),
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_terminal BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on ticket_type_id for type-scoped queries
CREATE INDEX idx_status_definition_ticket_type_id ON status_definition(ticket_type_id);

-- Index on ticket_type_id + display_order for ordered listing
CREATE INDEX idx_status_definition_ticket_type_id_display_order ON status_definition(ticket_type_id, display_order);

-- Ensure exactly one default status per ticket type
CREATE UNIQUE INDEX idx_status_definition_one_default_per_type
    ON status_definition(ticket_type_id)
    WHERE is_default = true;

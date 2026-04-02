-- Create ticket_type table
CREATE TABLE ticket_type (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(255),
    color VARCHAR(7),
    default_sop_template_id INT REFERENCES sop_template(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on space_id for space-scoped queries
CREATE INDEX idx_ticket_type_space_id ON ticket_type(space_id);

-- Index on space_id + display_order for ordered listing
CREATE INDEX idx_ticket_type_space_id_display_order ON ticket_type(space_id, display_order);

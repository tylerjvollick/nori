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

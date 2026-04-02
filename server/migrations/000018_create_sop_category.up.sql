-- Create sop_category table for hierarchical SOP organization
CREATE TABLE sop_category (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_category_id UUID REFERENCES sop_category(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on space_id for space-scoped queries
CREATE INDEX idx_sop_category_space_id ON sop_category(space_id);

-- Index on parent_category_id for tree traversal
CREATE INDEX idx_sop_category_parent_id ON sop_category(parent_category_id);

-- Index for ordered listing within a parent
CREATE INDEX idx_sop_category_space_parent_order ON sop_category(space_id, parent_category_id, display_order);

-- Add sop_category_id column to sop_template for folder organization
ALTER TABLE sop_template ADD COLUMN sop_category_id UUID REFERENCES sop_category(id) ON DELETE SET NULL;

-- Index on sop_template.sop_category_id for category-based queries
CREATE INDEX idx_sop_template_sop_category_id ON sop_template(sop_category_id);

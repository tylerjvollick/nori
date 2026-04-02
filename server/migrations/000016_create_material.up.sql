-- Create material table for inventory tracking
CREATE TABLE material (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'other',
    unit VARCHAR(50) NOT NULL,
    current_stock NUMERIC(12,4) NOT NULL DEFAULT 0,
    reorder_threshold NUMERIC(12,4) NOT NULL DEFAULT 0,
    reorder_quantity NUMERIC(12,4) NOT NULL DEFAULT 0,
    location VARCHAR(255),
    unit_cost NUMERIC(12,4),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Constrain category to valid enum values
ALTER TABLE material ADD CONSTRAINT chk_material_category
    CHECK (category IN ('lumber', 'hardware', 'finish', 'consumable', 'other'));

-- Index on space_id for space-scoped queries
CREATE INDEX idx_material_space_id ON material(space_id);

-- Index on space_id + name for searching materials within a space
CREATE INDEX idx_material_space_id_name ON material(space_id, name);

-- Index on space_id + category for filtering by category
CREATE INDEX idx_material_space_id_category ON material(space_id, category);

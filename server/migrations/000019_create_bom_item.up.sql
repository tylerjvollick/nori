-- Create bom_item table for bill of materials line items on recipe versions.
-- FK to recipe_version is added in 000038 after the recipe tables exist.
CREATE TABLE bom_item (
    id SERIAL PRIMARY KEY,
    recipe_version_id INT NOT NULL,
    material_id UUID REFERENCES material(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    quantity NUMERIC(12,4) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    unit_cost NUMERIC(12,4),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on recipe_version_id for version-scoped queries
CREATE INDEX idx_bom_item_recipe_version_id ON bom_item(recipe_version_id);

-- Index on material_id for material usage lookups
CREATE INDEX idx_bom_item_material_id ON bom_item(material_id);

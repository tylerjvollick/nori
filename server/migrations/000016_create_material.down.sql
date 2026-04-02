-- Remove indexes
DROP INDEX IF EXISTS idx_material_space_id_category;
DROP INDEX IF EXISTS idx_material_space_id_name;
DROP INDEX IF EXISTS idx_material_space_id;

-- Drop material table
DROP TABLE IF EXISTS material;

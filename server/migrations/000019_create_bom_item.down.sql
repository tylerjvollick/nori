-- Remove bom_item table
DROP INDEX IF EXISTS idx_bom_item_material_id;
DROP INDEX IF EXISTS idx_bom_item_recipe_version_id;
DROP TABLE IF EXISTS bom_item;

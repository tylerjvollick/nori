-- Remove sop_category_id from sop_template
DROP INDEX IF EXISTS idx_sop_template_sop_category_id;
ALTER TABLE sop_template DROP COLUMN IF EXISTS sop_category_id;

-- Remove sop_category table
DROP INDEX IF EXISTS idx_sop_category_space_parent_order;
DROP INDEX IF EXISTS idx_sop_category_parent_id;
DROP INDEX IF EXISTS idx_sop_category_space_id;
DROP TABLE IF EXISTS sop_category;

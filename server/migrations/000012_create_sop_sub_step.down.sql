-- Remove indexes
DROP INDEX IF EXISTS idx_sop_sub_step_step_order;
DROP INDEX IF EXISTS idx_sop_sub_step_sop_step_id;

-- Drop table
DROP TABLE IF EXISTS sop_sub_step;

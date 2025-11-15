-- Drop indexes
DROP INDEX IF EXISTS idx_sop_step_photo_uuid;
DROP INDEX IF EXISTS idx_sop_step_photo_order;
DROP INDEX IF EXISTS idx_sop_step_photo_step_id;

-- Drop table
DROP TABLE IF EXISTS sop_step_photo;

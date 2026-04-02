-- Remove check constraint
ALTER TABLE sop_step_media DROP CONSTRAINT IF EXISTS chk_sop_step_media_owner;

-- Remove index on sop_sub_step_id
DROP INDEX IF EXISTS idx_sop_step_media_sop_sub_step_id;

-- Remove sop_sub_step_id column
ALTER TABLE sop_step_media DROP COLUMN IF EXISTS sop_sub_step_id;

-- Restore sop_step_id to NOT NULL (backfill any nulls first)
UPDATE sop_step_media SET sop_step_id = 0 WHERE sop_step_id IS NULL;
ALTER TABLE sop_step_media ALTER COLUMN sop_step_id SET NOT NULL;

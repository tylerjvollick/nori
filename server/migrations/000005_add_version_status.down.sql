-- Remove indexes
DROP INDEX IF EXISTS idx_sop_template_version_user_status;
DROP INDEX IF EXISTS idx_sop_template_version_status;

-- Remove constraint
ALTER TABLE sop_template_version
DROP CONSTRAINT IF EXISTS chk_version_status;

-- Remove status column
ALTER TABLE sop_template_version
DROP COLUMN IF EXISTS status;

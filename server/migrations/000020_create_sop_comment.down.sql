-- Remove indexes
DROP INDEX IF EXISTS idx_sop_comment_user_id;
DROP INDEX IF EXISTS idx_sop_comment_sop_sub_step_id;
DROP INDEX IF EXISTS idx_sop_comment_sop_step_id;
DROP INDEX IF EXISTS idx_sop_comment_sop_template_id;

-- Drop sop_comment table
DROP TABLE IF EXISTS sop_comment;

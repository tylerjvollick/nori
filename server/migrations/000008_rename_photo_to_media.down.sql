-- Rollback: Rename sop_step_media back to sop_step_photo
ALTER TABLE sop_step_media DROP COLUMN IF EXISTS duration;

ALTER TABLE sop_step_media RENAME TO sop_step_photo;

-- Rename indexes back
ALTER INDEX idx_sop_step_media_step_id RENAME TO idx_sop_step_photo_step_id;
ALTER INDEX idx_sop_step_media_order RENAME TO idx_sop_step_photo_order;
ALTER INDEX idx_sop_step_media_uuid RENAME TO idx_sop_step_photo_uuid;

-- Update table comment
COMMENT ON TABLE sop_step_photo IS 'Photos attached to SOP steps with lexicographic ordering';

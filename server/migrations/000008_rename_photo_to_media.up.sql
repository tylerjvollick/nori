-- Rename sop_step_photo table to sop_step_media to support videos
ALTER TABLE sop_step_photo RENAME TO sop_step_media;

-- Rename indexes
ALTER INDEX idx_sop_step_photo_step_id RENAME TO idx_sop_step_media_step_id;
ALTER INDEX idx_sop_step_photo_order RENAME TO idx_sop_step_media_order;
ALTER INDEX idx_sop_step_photo_uuid RENAME TO idx_sop_step_media_uuid;

-- Update table comment
COMMENT ON TABLE sop_step_media IS 'Media (photos and videos) attached to SOP steps with lexicographic ordering';

-- Optional: Add duration field for videos (in seconds)
ALTER TABLE sop_step_media ADD COLUMN duration INTEGER;

COMMENT ON COLUMN sop_step_media.duration IS 'Duration in seconds (for video files only)';

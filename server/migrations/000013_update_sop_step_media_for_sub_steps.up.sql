-- Make sop_step_id nullable (was NOT NULL)
ALTER TABLE sop_step_media ALTER COLUMN sop_step_id DROP NOT NULL;

-- Add sop_sub_step_id column (nullable, FK to sop_sub_step)
ALTER TABLE sop_step_media
    ADD COLUMN sop_sub_step_id INTEGER REFERENCES sop_sub_step(id) ON DELETE CASCADE;

-- Add index on sop_sub_step_id
CREATE INDEX idx_sop_step_media_sop_sub_step_id ON sop_step_media(sop_sub_step_id);

-- Add check constraint: exactly one of sop_step_id or sop_sub_step_id must be set
ALTER TABLE sop_step_media
    ADD CONSTRAINT chk_sop_step_media_owner
    CHECK (
        (sop_step_id IS NOT NULL AND sop_sub_step_id IS NULL) OR
        (sop_step_id IS NULL AND sop_sub_step_id IS NOT NULL)
    );

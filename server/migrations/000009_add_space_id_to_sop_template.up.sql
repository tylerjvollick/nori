-- Add space_id column to sop_template
ALTER TABLE sop_template
    ADD COLUMN space_id UUID;

-- Add foreign key constraint
ALTER TABLE sop_template
    ADD CONSTRAINT fk_sop_template_space
    FOREIGN KEY (space_id)
    REFERENCES spaces (id)
    ON DELETE SET NULL;

-- Create index for scoping queries by space
CREATE INDEX IF NOT EXISTS idx_sop_template_space_id
    ON sop_template (space_id);

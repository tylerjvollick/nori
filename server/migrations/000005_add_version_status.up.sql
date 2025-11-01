-- Add status column to sop_template_version table
ALTER TABLE sop_template_version 
ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'published';

-- Add check constraint to ensure valid statuses
ALTER TABLE sop_template_version
ADD CONSTRAINT chk_version_status CHECK (status IN ('draft', 'published'));

-- Create index for efficient draft queries
CREATE INDEX idx_sop_template_version_status ON sop_template_version (status);

-- Create index for efficient user draft queries
CREATE INDEX idx_sop_template_version_user_status ON sop_template_version (created_by, status);

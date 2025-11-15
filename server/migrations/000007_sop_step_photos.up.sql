-- Create sop_step_photo table for storing photos attached to SOP steps
CREATE TABLE sop_step_photo (
    id SERIAL PRIMARY KEY,
    sop_step_id INTEGER NOT NULL REFERENCES sop_step(id) ON DELETE CASCADE,
    uuid VARCHAR(255) NOT NULL UNIQUE,
    file_path VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    "order" VARCHAR(50) NOT NULL DEFAULT 'a',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Add indexes for efficient queries
CREATE INDEX idx_sop_step_photo_step_id ON sop_step_photo(sop_step_id);
CREATE INDEX idx_sop_step_photo_order ON sop_step_photo("order");
CREATE INDEX idx_sop_step_photo_uuid ON sop_step_photo(uuid);

-- Add comment to table
COMMENT ON TABLE sop_step_photo IS 'Photos attached to SOP steps with lexicographic ordering';

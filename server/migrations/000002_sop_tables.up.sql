-- =========================================
-- SOP TEMPLATE (Versioned)
-- =========================================
CREATE TABLE IF NOT EXISTS sop_template (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    current_version_id INT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_sop_template_created_by
        FOREIGN KEY (created_by)
        REFERENCES users (id)
        ON DELETE CASCADE
);

-- =========================================
-- SOP TEMPLATE VERSIONS
-- =========================================
-- Each change to structure or description creates a new row here.
CREATE TABLE IF NOT EXISTS sop_template_version (
    id SERIAL PRIMARY KEY,
    sop_template_id INT NOT NULL,
    version_number INT NOT NULL,
    description TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    change_summary TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    CONSTRAINT fk_sop_template_version_template
        FOREIGN KEY (sop_template_id)
        REFERENCES sop_template (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_sop_template_version_created_by
        FOREIGN KEY (created_by)
        REFERENCES users (id)
        ON DELETE CASCADE,
    UNIQUE (sop_template_id, version_number)
);

-- =========================================
-- SOP STEPS (Belong to specific versions)
-- =========================================
-- Steps are versioned implicitly through their parent sop_template_version.
CREATE TABLE IF NOT EXISTS sop_step (
    id SERIAL PRIMARY KEY,
    sop_template_version_id INT NOT NULL,
    step_number INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    instructions TEXT,
    estimated_time_minutes INT,
    image_url TEXT,
    video_url TEXT,
    requires_approval BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_sop_step_version
        FOREIGN KEY (sop_template_version_id)
        REFERENCES sop_template_version (id)
        ON DELETE CASCADE,
    UNIQUE (sop_template_version_id, step_number)
);

-- Add foreign key from sop_template to sop_template_version
-- This must be added after both tables exist
ALTER TABLE sop_template
    ADD CONSTRAINT fk_sop_template_current_version
    FOREIGN KEY (current_version_id)
    REFERENCES sop_template_version (id)
    ON DELETE SET NULL;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_sop_template_created_by ON sop_template (created_by);
CREATE INDEX IF NOT EXISTS idx_sop_template_version_template ON sop_template_version (sop_template_id);
CREATE INDEX IF NOT EXISTS idx_sop_template_version_created_by ON sop_template_version (created_by);
CREATE INDEX IF NOT EXISTS idx_sop_step_version ON sop_step (sop_template_version_id);

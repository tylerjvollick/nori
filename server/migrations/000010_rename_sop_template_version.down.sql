-- =========================================
-- Reverse: sop_version → sop_template_version
-- Restore Status, Materials, Equipment columns
-- Rename sop_step FK column back
-- =========================================

-- 1. Rename the FK column on sop_step back
ALTER TABLE sop_step RENAME COLUMN sop_version_id TO sop_template_version_id;

-- 2. Update FK constraint on sop_step
ALTER TABLE sop_step DROP CONSTRAINT IF EXISTS fk_sop_step_version;
ALTER TABLE sop_step ADD CONSTRAINT fk_sop_step_version
    FOREIGN KEY (sop_template_version_id) REFERENCES sop_version (id) ON DELETE CASCADE;

-- 3. Rename the table back
ALTER TABLE sop_version RENAME TO sop_template_version;

-- 4. Update FK constraint on sop_template
ALTER TABLE sop_template DROP CONSTRAINT IF EXISTS fk_sop_template_current_version;
ALTER TABLE sop_template ADD CONSTRAINT fk_sop_template_current_version
    FOREIGN KEY (current_version_id) REFERENCES sop_template_version (id) ON DELETE SET NULL;

-- 5. Restore dropped columns
ALTER TABLE sop_template_version ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'published';
ALTER TABLE sop_template_version ADD COLUMN IF NOT EXISTS materials TEXT[];
ALTER TABLE sop_template_version ADD COLUMN IF NOT EXISTS equipment TEXT[];

-- 6. Restore indexes
DROP INDEX IF EXISTS idx_sop_version_template;
CREATE INDEX idx_sop_template_version_template ON sop_template_version (sop_template_id);

DROP INDEX IF EXISTS idx_sop_version_created_by;
CREATE INDEX idx_sop_template_version_created_by ON sop_template_version (created_by);

DROP INDEX IF EXISTS idx_sop_step_version;
CREATE INDEX idx_sop_step_version ON sop_step (sop_template_version_id);

-- 7. Restore unique constraint name
ALTER TABLE sop_template_version DROP CONSTRAINT IF EXISTS sop_version_sop_template_id_version_number_key;
ALTER TABLE sop_template_version ADD CONSTRAINT sop_template_version_sop_template_id_version_number_key
    UNIQUE (sop_template_id, version_number);

-- 8. Update FK constraint on sop_step to reference original table name
ALTER TABLE sop_step DROP CONSTRAINT IF EXISTS fk_sop_step_version;
ALTER TABLE sop_step ADD CONSTRAINT fk_sop_step_version
    FOREIGN KEY (sop_template_version_id) REFERENCES sop_template_version (id) ON DELETE CASCADE;

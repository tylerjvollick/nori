-- =========================================
-- Rename sop_template_version → sop_version
-- Drop Status, Materials, Equipment columns
-- Rename sop_step FK column
-- =========================================

-- 1. Rename the table
ALTER TABLE sop_template_version RENAME TO sop_version;

-- 2. Drop the Status column (draft/published replaced by auto-versioning)
ALTER TABLE sop_version DROP COLUMN IF EXISTS status;

-- 3. Drop the Materials and Equipment columns (replaced by BOMItem)
ALTER TABLE sop_version DROP COLUMN IF EXISTS materials;
ALTER TABLE sop_version DROP COLUMN IF EXISTS equipment;

-- 4. Rename the FK column on sop_step
ALTER TABLE sop_step RENAME COLUMN sop_template_version_id TO sop_version_id;

-- 5. Update FK constraint on sop_step to reference renamed table
ALTER TABLE sop_step DROP CONSTRAINT IF EXISTS fk_sop_step_version;
ALTER TABLE sop_step ADD CONSTRAINT fk_sop_step_version
    FOREIGN KEY (sop_version_id) REFERENCES sop_version (id) ON DELETE CASCADE;

-- 6. Update FK constraint on sop_template.current_version_id
ALTER TABLE sop_template DROP CONSTRAINT IF EXISTS fk_sop_template_current_version;
ALTER TABLE sop_template ADD CONSTRAINT fk_sop_template_current_version
    FOREIGN KEY (current_version_id) REFERENCES sop_version (id) ON DELETE SET NULL;

-- 7. Rename indexes to reflect new table name
DROP INDEX IF EXISTS idx_sop_template_version_template;
CREATE INDEX idx_sop_version_template ON sop_version (sop_template_id);

DROP INDEX IF EXISTS idx_sop_template_version_created_by;
CREATE INDEX idx_sop_version_created_by ON sop_version (created_by);

DROP INDEX IF EXISTS idx_sop_step_version;
CREATE INDEX idx_sop_step_version ON sop_step (sop_version_id);

-- 8. Drop the unique constraint and recreate with correct table reference
ALTER TABLE sop_version DROP CONSTRAINT IF EXISTS sop_template_version_sop_template_id_version_number_key;
ALTER TABLE sop_version ADD CONSTRAINT sop_version_sop_template_id_version_number_key
    UNIQUE (sop_template_id, version_number);

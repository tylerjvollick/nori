-- =========================================
-- Remove materials and equipment from SOP Template Version
-- =========================================
ALTER TABLE sop_template_version
    DROP COLUMN IF EXISTS materials,
    DROP COLUMN IF EXISTS equipment;

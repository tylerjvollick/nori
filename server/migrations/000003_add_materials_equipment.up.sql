-- =========================================
-- Add materials and equipment to SOP Template Version
-- =========================================
ALTER TABLE sop_template_version
    ADD COLUMN materials TEXT[],
    ADD COLUMN equipment TEXT[];

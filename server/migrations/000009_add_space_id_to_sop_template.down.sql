-- Remove index
DROP INDEX IF EXISTS idx_sop_template_space_id;

-- Remove foreign key constraint
ALTER TABLE sop_template
    DROP CONSTRAINT IF EXISTS fk_sop_template_space;

-- Remove space_id column
ALTER TABLE sop_template
    DROP COLUMN IF EXISTS space_id;

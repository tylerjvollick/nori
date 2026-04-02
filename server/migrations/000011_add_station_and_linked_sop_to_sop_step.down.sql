-- Remove indexes
DROP INDEX IF EXISTS idx_sop_step_linked_sop_template_id;
DROP INDEX IF EXISTS idx_sop_step_station_id;

-- Remove foreign key constraint
ALTER TABLE sop_step
    DROP CONSTRAINT IF EXISTS fk_sop_step_linked_sop_template;

-- Remove columns
ALTER TABLE sop_step
    DROP COLUMN IF EXISTS linked_sop_template_id;

ALTER TABLE sop_step
    DROP COLUMN IF EXISTS station_id;

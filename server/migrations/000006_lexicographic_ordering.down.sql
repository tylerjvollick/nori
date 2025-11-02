-- Restore step_number if it was dropped
-- ALTER TABLE sop_step ADD COLUMN step_number INTEGER;

-- Migrate order back to step_number if needed
-- This is a lossy operation as lexicographic order doesn't map perfectly back

-- Remove index
DROP INDEX IF EXISTS idx_sop_step_order;

-- Remove order column
ALTER TABLE sop_step DROP COLUMN "order";

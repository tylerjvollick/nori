-- Add order column to sop_step table
ALTER TABLE sop_step ADD COLUMN "order" VARCHAR(50) NOT NULL DEFAULT 'a';

-- Migrate existing data: convert step_number to lexicographic order
-- For existing steps, generate order values based on step_number
UPDATE sop_step 
SET "order" = CASE 
    WHEN step_number <= 26 THEN chr(96 + step_number)  -- 'a' through 'z'
    WHEN step_number <= 52 THEN 'a' || chr(70 + step_number) -- 'aa' through 'az'
    WHEN step_number <= 78 THEN 'b' || chr(44 + step_number) -- 'ba' through 'bz'
    ELSE 'c' || chr(18 + step_number) -- fallback for larger numbers
END;

-- Add index on order column for efficient sorting
CREATE INDEX idx_sop_step_order ON sop_step("order");

-- Remove step_number column
ALTER TABLE sop_step DROP COLUMN step_number;

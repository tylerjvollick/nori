-- Remove default_labor_rate column from spaces table.

ALTER TABLE spaces DROP COLUMN IF EXISTS default_labor_rate;

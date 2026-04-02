-- Add default_labor_rate column to spaces table.
-- Space-level default hourly labor rate used by the cost computation service
-- to auto-generate labor CostEntries from TimeEvents.

ALTER TABLE spaces ADD COLUMN default_labor_rate NUMERIC(12,4);

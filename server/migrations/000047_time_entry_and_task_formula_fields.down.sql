DROP TABLE IF EXISTS time_entry;

ALTER TABLE task DROP COLUMN IF EXISTS estimated_time_from_recipe_secs;
ALTER TABLE task DROP COLUMN IF EXISTS estimated_time_formula;

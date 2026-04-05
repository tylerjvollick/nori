-- Remove foreign keys from task table
ALTER TABLE task DROP CONSTRAINT IF EXISTS fk_task_recipe;
ALTER TABLE task DROP CONSTRAINT IF EXISTS fk_task_recipe_version;

-- Drop recipe_version table (also drops the FK from recipe.current_version_id)
ALTER TABLE recipe DROP CONSTRAINT IF EXISTS fk_recipe_current_version;
DROP TABLE IF EXISTS recipe_version;

-- Drop recipe table
DROP TABLE IF EXISTS recipe;

-- Drop recipe_version_status enum
DROP TYPE IF EXISTS recipe_version_status;

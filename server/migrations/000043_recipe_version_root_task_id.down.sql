-- Remove index
DROP INDEX IF EXISTS idx_recipe_version_root_task_id;

-- Restore content to NOT NULL (backfill empty string for any NULLs)
UPDATE recipe_version SET content = '' WHERE content IS NULL;
ALTER TABLE recipe_version ALTER COLUMN content SET NOT NULL;

-- Remove root_task_id column
ALTER TABLE recipe_version DROP COLUMN IF EXISTS root_task_id;

-- Add root_task_id to recipe_version, pointing to the frozen task tree for this version
ALTER TABLE recipe_version ADD COLUMN root_task_id UUID REFERENCES task(id);

-- Make content nullable (keep for backward compat, stop writing to it)
ALTER TABLE recipe_version ALTER COLUMN content DROP NOT NULL;

-- Index on root_task_id for lookups
CREATE INDEX idx_recipe_version_root_task_id ON recipe_version(root_task_id);

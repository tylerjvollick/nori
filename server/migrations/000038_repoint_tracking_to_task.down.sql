-- Rollback: Remove deferred foreign keys and tag join tables.

BEGIN;

-- Drop tag join tables
DROP TABLE IF EXISTS recipe_tag;
DROP TABLE IF EXISTS task_tag;

-- Drop deferred FKs
ALTER TABLE bom_item DROP CONSTRAINT IF EXISTS bom_item_recipe_version_id_fkey;
ALTER TABLE cost_entry DROP CONSTRAINT IF EXISTS cost_entry_task_id_fkey;
ALTER TABLE time_event DROP CONSTRAINT IF EXISTS time_event_task_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_step_task_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_linked_task_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_task_id_fkey;

COMMIT;

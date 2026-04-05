-- Migration: Re-point tracking models from Ticket/SOP to Task/Recipe
-- activity_entry, time_event, cost_entry, bom_item, tag join tables

BEGIN;

-- ============================================================
-- 1. activity_entry: ticket_id → task_id, ticket_step_id → step_task_id,
--    linked_ticket_id → linked_task_id
-- ============================================================

-- Drop old indexes
DROP INDEX IF EXISTS idx_activity_entry_ticket;
DROP INDEX IF EXISTS idx_activity_entry_ticket_chronological;
DROP INDEX IF EXISTS idx_activity_entry_ticket_step;

-- Drop old FK constraints (if they exist)
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_ticket_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_linked_ticket_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_ticket_step_id_fkey;

-- Drop old columns
ALTER TABLE activity_entry DROP COLUMN IF EXISTS ticket_id;
ALTER TABLE activity_entry DROP COLUMN IF EXISTS linked_ticket_id;
ALTER TABLE activity_entry DROP COLUMN IF EXISTS ticket_step_id;

-- Add new columns
ALTER TABLE activity_entry ADD COLUMN task_id VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE activity_entry ADD COLUMN linked_task_id VARCHAR(255);
ALTER TABLE activity_entry ADD COLUMN step_task_id VARCHAR(255);

-- Remove the default after adding (was only for existing rows)
ALTER TABLE activity_entry ALTER COLUMN task_id DROP DEFAULT;

-- Add FK constraints
ALTER TABLE activity_entry ADD CONSTRAINT activity_entry_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES task(id) ON DELETE CASCADE;
ALTER TABLE activity_entry ADD CONSTRAINT activity_entry_linked_task_id_fkey
    FOREIGN KEY (linked_task_id) REFERENCES task(id) ON DELETE SET NULL;
ALTER TABLE activity_entry ADD CONSTRAINT activity_entry_step_task_id_fkey
    FOREIGN KEY (step_task_id) REFERENCES task(id) ON DELETE SET NULL;

-- New indexes
CREATE INDEX idx_activity_entry_task ON activity_entry(task_id);
CREATE INDEX idx_activity_entry_task_chronological ON activity_entry(task_id, created_at ASC);
CREATE INDEX idx_activity_entry_step_task ON activity_entry(step_task_id) WHERE step_task_id IS NOT NULL;

-- ============================================================
-- 2. time_event: add task_id (nullable), drop ticket_id & ticket_step_id
-- ============================================================

-- Drop old indexes
DROP INDEX IF EXISTS idx_time_event_ticket;
DROP INDEX IF EXISTS idx_time_event_ticket_step;

-- Drop old FK constraints
ALTER TABLE time_event DROP CONSTRAINT IF EXISTS time_event_ticket_id_fkey;
ALTER TABLE time_event DROP CONSTRAINT IF EXISTS time_event_ticket_step_id_fkey;

-- Drop old columns
ALTER TABLE time_event DROP COLUMN IF EXISTS ticket_id;
ALTER TABLE time_event DROP COLUMN IF EXISTS ticket_step_id;

-- Add new column
ALTER TABLE time_event ADD COLUMN task_id VARCHAR(255);

-- Add FK constraint
ALTER TABLE time_event ADD CONSTRAINT time_event_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES task(id) ON DELETE SET NULL;

-- New index
CREATE INDEX idx_time_event_task ON time_event(task_id) WHERE task_id IS NOT NULL;

-- ============================================================
-- 3. cost_entry: ticket_id → task_id
-- ============================================================

-- Drop old index
DROP INDEX IF EXISTS idx_cost_entry_ticket;
DROP INDEX IF EXISTS idx_cost_entry_cost_type;

-- Drop old FK constraint
ALTER TABLE cost_entry DROP CONSTRAINT IF EXISTS cost_entry_ticket_id_fkey;

-- Drop old column
ALTER TABLE cost_entry DROP COLUMN IF EXISTS ticket_id;

-- Add new column
ALTER TABLE cost_entry ADD COLUMN task_id VARCHAR(255) NOT NULL DEFAULT '';

-- Remove the default after adding
ALTER TABLE cost_entry ALTER COLUMN task_id DROP DEFAULT;

-- Add FK constraint
ALTER TABLE cost_entry ADD CONSTRAINT cost_entry_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES task(id) ON DELETE CASCADE;

-- New indexes
CREATE INDEX idx_cost_entry_task ON cost_entry(task_id);
CREATE INDEX idx_cost_entry_task_cost_type ON cost_entry(task_id, cost_type);

-- ============================================================
-- 4. bom_item: sop_version_id → recipe_version_id, drop sop_step_id
-- ============================================================

-- Drop old indexes
DROP INDEX IF EXISTS idx_bom_item_sop_version_id;
DROP INDEX IF EXISTS idx_bom_item_sop_step_id;

-- Drop old FK constraints
ALTER TABLE bom_item DROP CONSTRAINT IF EXISTS bom_item_sop_version_id_fkey;
ALTER TABLE bom_item DROP CONSTRAINT IF EXISTS bom_item_sop_step_id_fkey;

-- Drop old columns
ALTER TABLE bom_item DROP COLUMN IF EXISTS sop_version_id;
ALTER TABLE bom_item DROP COLUMN IF EXISTS sop_step_id;

-- Add new column
ALTER TABLE bom_item ADD COLUMN recipe_version_id INT NOT NULL DEFAULT 0;

-- Remove the default after adding
ALTER TABLE bom_item ALTER COLUMN recipe_version_id DROP DEFAULT;

-- Add FK constraint
ALTER TABLE bom_item ADD CONSTRAINT bom_item_recipe_version_id_fkey
    FOREIGN KEY (recipe_version_id) REFERENCES recipe_version(id) ON DELETE CASCADE;

-- New index
CREATE INDEX idx_bom_item_recipe_version_id ON bom_item(recipe_version_id);

-- ============================================================
-- 5. Tag join tables: replace sop_template_tag with recipe_tag,
--    replace ticket_tag with task_tag
-- ============================================================

-- Drop old join tables
DROP TABLE IF EXISTS sop_template_tag;
DROP TABLE IF EXISTS ticket_tag;

-- Create task_tag join table
CREATE TABLE task_tag (
    task_id VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX idx_task_tag_task_id ON task_tag(task_id);
CREATE INDEX idx_task_tag_tag_id ON task_tag(tag_id);

-- Create recipe_tag join table
CREATE TABLE recipe_tag (
    recipe_id UUID NOT NULL REFERENCES recipe(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (recipe_id, tag_id)
);

CREATE INDEX idx_recipe_tag_recipe_id ON recipe_tag(recipe_id);
CREATE INDEX idx_recipe_tag_tag_id ON recipe_tag(tag_id);

COMMIT;

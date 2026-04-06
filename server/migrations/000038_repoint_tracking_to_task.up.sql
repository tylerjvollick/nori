-- Migration: Add deferred foreign keys and create tag join tables.
-- These FKs were deferred because the referenced tables (task, recipe,
-- recipe_version) are created in later migrations (000035, 000037).

BEGIN;

-- ============================================================
-- 1. activity_entry: add FK to task
-- ============================================================
ALTER TABLE activity_entry ADD CONSTRAINT activity_entry_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES task(id) ON DELETE CASCADE;
ALTER TABLE activity_entry ADD CONSTRAINT activity_entry_linked_task_id_fkey
    FOREIGN KEY (linked_task_id) REFERENCES task(id) ON DELETE SET NULL;
ALTER TABLE activity_entry ADD CONSTRAINT activity_entry_step_task_id_fkey
    FOREIGN KEY (step_task_id) REFERENCES task(id) ON DELETE SET NULL;

-- ============================================================
-- 2. time_event: add FK to task
-- ============================================================
ALTER TABLE time_event ADD CONSTRAINT time_event_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES task(id) ON DELETE SET NULL;

-- ============================================================
-- 3. cost_entry: add FK to task
-- ============================================================
ALTER TABLE cost_entry ADD CONSTRAINT cost_entry_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES task(id) ON DELETE CASCADE;

-- ============================================================
-- 4. bom_item: add FK to recipe_version
-- ============================================================
ALTER TABLE bom_item ADD CONSTRAINT bom_item_recipe_version_id_fkey
    FOREIGN KEY (recipe_version_id) REFERENCES recipe_version(id) ON DELETE CASCADE;

-- ============================================================
-- 5. Tag join tables: task_tag and recipe_tag
-- ============================================================

CREATE TABLE task_tag (
    task_id VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX idx_task_tag_task_id ON task_tag(task_id);
CREATE INDEX idx_task_tag_tag_id ON task_tag(tag_id);

CREATE TABLE recipe_tag (
    recipe_id UUID NOT NULL REFERENCES recipe(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (recipe_id, tag_id)
);

CREATE INDEX idx_recipe_tag_recipe_id ON recipe_tag(recipe_id);
CREATE INDEX idx_recipe_tag_tag_id ON recipe_tag(tag_id);

COMMIT;

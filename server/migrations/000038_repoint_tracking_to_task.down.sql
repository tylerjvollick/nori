-- Rollback: Revert tracking models from Task/Recipe back to Ticket/SOP

BEGIN;

-- ============================================================
-- 5. Tag join tables: revert to sop_template_tag and ticket_tag
-- ============================================================

DROP TABLE IF EXISTS recipe_tag;
DROP TABLE IF EXISTS task_tag;

-- Recreate sop_template_tag (FK to sop_template may not exist; skip FK)
CREATE TABLE sop_template_tag (
    sop_template_id INT NOT NULL,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (sop_template_id, tag_id)
);

CREATE INDEX idx_sop_template_tag_template_id ON sop_template_tag(sop_template_id);
CREATE INDEX idx_sop_template_tag_tag_id ON sop_template_tag(tag_id);

-- Recreate ticket_tag (FK to ticket may not exist; skip FK)
CREATE TABLE ticket_tag (
    ticket_id UUID NOT NULL,
    tag_id UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (ticket_id, tag_id)
);

CREATE INDEX idx_ticket_tag_ticket_id ON ticket_tag(ticket_id);
CREATE INDEX idx_ticket_tag_tag_id ON ticket_tag(tag_id);

-- ============================================================
-- 4. bom_item: recipe_version_id → sop_version_id, re-add sop_step_id
-- ============================================================

DROP INDEX IF EXISTS idx_bom_item_recipe_version_id;

ALTER TABLE bom_item DROP CONSTRAINT IF EXISTS bom_item_recipe_version_id_fkey;
ALTER TABLE bom_item DROP COLUMN IF EXISTS recipe_version_id;

ALTER TABLE bom_item ADD COLUMN sop_version_id INT NOT NULL DEFAULT 0;
ALTER TABLE bom_item ALTER COLUMN sop_version_id DROP DEFAULT;
ALTER TABLE bom_item ADD COLUMN sop_step_id INT;

CREATE INDEX idx_bom_item_sop_version_id ON bom_item(sop_version_id);
CREATE INDEX idx_bom_item_sop_step_id ON bom_item(sop_step_id);

-- ============================================================
-- 3. cost_entry: task_id → ticket_id
-- ============================================================

DROP INDEX IF EXISTS idx_cost_entry_task;
DROP INDEX IF EXISTS idx_cost_entry_task_cost_type;

ALTER TABLE cost_entry DROP CONSTRAINT IF EXISTS cost_entry_task_id_fkey;
ALTER TABLE cost_entry DROP COLUMN IF EXISTS task_id;

ALTER TABLE cost_entry ADD COLUMN ticket_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
ALTER TABLE cost_entry ALTER COLUMN ticket_id DROP DEFAULT;

CREATE INDEX idx_cost_entry_ticket ON cost_entry(ticket_id);
CREATE INDEX idx_cost_entry_cost_type ON cost_entry(ticket_id, cost_type);

-- ============================================================
-- 2. time_event: task_id → ticket_id & ticket_step_id
-- ============================================================

DROP INDEX IF EXISTS idx_time_event_task;

ALTER TABLE time_event DROP CONSTRAINT IF EXISTS time_event_task_id_fkey;
ALTER TABLE time_event DROP COLUMN IF EXISTS task_id;

ALTER TABLE time_event ADD COLUMN ticket_id UUID;
ALTER TABLE time_event ADD COLUMN ticket_step_id UUID;

CREATE INDEX idx_time_event_ticket ON time_event(ticket_id) WHERE ticket_id IS NOT NULL;
CREATE INDEX idx_time_event_ticket_step ON time_event(ticket_step_id) WHERE ticket_step_id IS NOT NULL;

-- ============================================================
-- 1. activity_entry: task_id → ticket_id, step_task_id → ticket_step_id,
--    linked_task_id → linked_ticket_id
-- ============================================================

DROP INDEX IF EXISTS idx_activity_entry_task;
DROP INDEX IF EXISTS idx_activity_entry_task_chronological;
DROP INDEX IF EXISTS idx_activity_entry_step_task;

ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_task_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_linked_task_id_fkey;
ALTER TABLE activity_entry DROP CONSTRAINT IF EXISTS activity_entry_step_task_id_fkey;

ALTER TABLE activity_entry DROP COLUMN IF EXISTS task_id;
ALTER TABLE activity_entry DROP COLUMN IF EXISTS linked_task_id;
ALTER TABLE activity_entry DROP COLUMN IF EXISTS step_task_id;

ALTER TABLE activity_entry ADD COLUMN ticket_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
ALTER TABLE activity_entry ALTER COLUMN ticket_id DROP DEFAULT;
ALTER TABLE activity_entry ADD COLUMN linked_ticket_id UUID;
ALTER TABLE activity_entry ADD COLUMN ticket_step_id UUID;

CREATE INDEX idx_activity_entry_ticket ON activity_entry(ticket_id);
CREATE INDEX idx_activity_entry_ticket_chronological ON activity_entry(ticket_id, created_at ASC);
CREATE INDEX idx_activity_entry_ticket_step ON activity_entry(ticket_step_id) WHERE ticket_step_id IS NOT NULL;

COMMIT;

-- Remove indexes
DROP INDEX IF EXISTS idx_status_definition_one_default_per_type;
DROP INDEX IF EXISTS idx_status_definition_ticket_type_id_display_order;
DROP INDEX IF EXISTS idx_status_definition_ticket_type_id;

-- Drop status_definition table
DROP TABLE IF EXISTS status_definition;

-- Drop status_category enum type
DROP TYPE IF EXISTS status_category;

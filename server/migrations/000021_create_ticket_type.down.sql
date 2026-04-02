-- Remove indexes
DROP INDEX IF EXISTS idx_ticket_type_space_id_display_order;
DROP INDEX IF EXISTS idx_ticket_type_space_id;

-- Drop ticket_type table
DROP TABLE IF EXISTS ticket_type;

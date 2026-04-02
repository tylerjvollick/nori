-- Drop indexes
DROP INDEX IF EXISTS idx_ticket_sub_step_order;
DROP INDEX IF EXISTS idx_ticket_sub_step_ticket_step_id;

-- Drop table
DROP TABLE IF EXISTS ticket_sub_step;

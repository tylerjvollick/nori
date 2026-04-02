-- Drop indexes
DROP INDEX IF EXISTS idx_ticket_step_order;
DROP INDEX IF EXISTS idx_ticket_step_status;
DROP INDEX IF EXISTS idx_ticket_step_assigned_to_id;
DROP INDEX IF EXISTS idx_ticket_step_station_id;
DROP INDEX IF EXISTS idx_ticket_step_ticket_id;

-- Drop table
DROP TABLE IF EXISTS ticket_step;

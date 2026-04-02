-- Drop indexes
DROP INDEX IF EXISTS idx_activity_entry_type;
DROP INDEX IF EXISTS idx_activity_entry_user;
DROP INDEX IF EXISTS idx_activity_entry_ticket_step;
DROP INDEX IF EXISTS idx_activity_entry_ticket_chronological;
DROP INDEX IF EXISTS idx_activity_entry_ticket;

-- Drop table
DROP TABLE IF EXISTS activity_entry;

-- Drop enum type
DROP TYPE IF EXISTS activity_entry_type;

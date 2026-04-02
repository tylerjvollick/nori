-- Drop indexes
DROP INDEX IF EXISTS idx_ticket_link_type;
DROP INDEX IF EXISTS idx_ticket_link_target;
DROP INDEX IF EXISTS idx_ticket_link_source;

-- Drop table
DROP TABLE IF EXISTS ticket_link;

-- Drop enum type
DROP TYPE IF EXISTS link_type;

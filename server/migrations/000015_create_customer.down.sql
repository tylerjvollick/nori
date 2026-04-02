-- Remove indexes
DROP INDEX IF EXISTS idx_customer_space_id_name;
DROP INDEX IF EXISTS idx_customer_space_id;

-- Drop customer table
DROP TABLE IF EXISTS customer;

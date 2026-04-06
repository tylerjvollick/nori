-- Remove quantity column from task table.

ALTER TABLE task DROP COLUMN IF EXISTS quantity;

-- Add quantity column to task table.
-- Stores how many pieces a task covers (e.g., "process 6 leg blanks").

ALTER TABLE task ADD COLUMN quantity INT NOT NULL DEFAULT 1;

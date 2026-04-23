-- Revert default task priority back to 0 (P0/Critical).
ALTER TABLE task ALTER COLUMN priority SET DEFAULT 0;

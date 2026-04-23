-- Change default task priority from 0 (P0/Critical) to 2 (P2/Medium).
ALTER TABLE task ALTER COLUMN priority SET DEFAULT 2;

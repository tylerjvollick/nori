-- Add 'recipe' to the task_type enum
ALTER TYPE task_type ADD VALUE IF NOT EXISTS 'recipe';

-- Add batch_size and estimated_time_secs columns to task table
ALTER TABLE task ADD COLUMN batch_size INT;
ALTER TABLE task ADD COLUMN estimated_time_secs INT;

-- Remove slug column from spaces table
DROP INDEX IF EXISTS idx_spaces_account_slug;
ALTER TABLE spaces DROP COLUMN IF EXISTS slug;

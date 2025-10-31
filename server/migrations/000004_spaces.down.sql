-- =========================================
-- Rollback spaces table and recent_spaces field
-- =========================================

-- Remove recent_spaces field from users
ALTER TABLE users
    DROP COLUMN IF EXISTS recent_spaces;

-- Drop indexes
DROP INDEX IF EXISTS idx_spaces_account_default;
DROP INDEX IF EXISTS idx_spaces_account_id;

-- Drop spaces table
DROP TABLE IF EXISTS spaces;

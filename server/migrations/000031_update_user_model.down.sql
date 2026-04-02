-- --------------------------------------------------------------------
-- Revert User model changes: restore GlobalRole and remove MustChangePassword
-- --------------------------------------------------------------------

-- Step 1: Convert role column back to text temporarily
ALTER TABLE users 
ALTER COLUMN role TYPE TEXT;

-- Step 2: Drop the new role enum type
DROP TYPE IF EXISTS role;

-- Step 3: Recreate the old global_role enum type
CREATE TYPE global_role AS ENUM ('ADMIN', 'USER', 'VIEWER');

-- Step 4: Map the new role values back to old format
UPDATE users 
SET role = CASE 
    WHEN role = 'admin' THEN 'ADMIN'
    WHEN role = 'user' THEN 'USER'
    ELSE 'USER'
END;

-- Step 5: Rename role column back to global_role
ALTER TABLE users 
RENAME COLUMN role TO global_role;

-- Step 6: Convert the column to use the old enum type
ALTER TABLE users 
ALTER COLUMN global_role TYPE global_role USING global_role::global_role;

-- Step 7: Remove must_change_password column
ALTER TABLE users 
DROP COLUMN must_change_password;

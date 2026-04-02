-- --------------------------------------------------------------------
-- Update User model: rename GlobalRole to Role and add MustChangePassword
-- --------------------------------------------------------------------

-- Step 1: Add the new must_change_password column
ALTER TABLE users 
ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT true;

-- Step 2: Rename global_role column to role
ALTER TABLE users 
RENAME COLUMN global_role TO role;

-- Step 3: Drop the old global_role enum type and create new role enum
-- First, we need to alter the column to not use the enum temporarily
ALTER TABLE users 
ALTER COLUMN role TYPE TEXT;

-- Drop the old enum type
DROP TYPE IF EXISTS global_role;

-- Create the new role enum type (admin, user - no viewer)
CREATE TYPE role AS ENUM ('admin', 'user');

-- Convert the column to use the new enum
-- Map old values to new values (ADMIN->admin, USER->user, VIEWER->user)
UPDATE users 
SET role = CASE 
    WHEN role = 'ADMIN' THEN 'admin'
    WHEN role = 'USER' THEN 'user'
    WHEN role = 'VIEWER' THEN 'user'
    ELSE 'user'
END;

-- Alter the column to use the new enum type
ALTER TABLE users 
ALTER COLUMN role TYPE role USING role::role;

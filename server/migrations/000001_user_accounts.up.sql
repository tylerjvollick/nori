-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- --------------------------------------------------------------------
-- Create enum type for GlobalRole
-- --------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'global_role') THEN
        CREATE TYPE global_role AS ENUM ('ADMIN', 'USER', 'VIEWER');
    END IF;
END
$$;

-- --------------------------------------------------------------------
-- Create users table
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    global_role global_role,
    email TEXT NOT NULL UNIQUE,
    password TEXT,
    first_name TEXT,
    last_name TEXT,
    default_account_id UUID
);

-- --------------------------------------------------------------------
-- Create accounts table
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT,
    sync_contact_and_billing_info BOOLEAN NOT NULL DEFAULT false,

    contact_first_name TEXT,
    contact_last_name TEXT,
    contact_phone_number TEXT,
    contact_phone_ext TEXT,
    contact_email TEXT,
    contact_address TEXT,
    contact_city TEXT,
    contact_state TEXT,
    contact_zip TEXT,

    billing_first_name TEXT,
    billing_last_name TEXT,
    billing_phone_number TEXT,
    billing_phone_ext TEXT,
    billing_email TEXT,
    billing_address TEXT,
    billing_city TEXT,
    billing_state TEXT,
    billing_zip TEXT,

    plan TEXT DEFAULT 'trial',

    created_by_user_id UUID NOT NULL,
    CONSTRAINT fk_accounts_created_by_user
        FOREIGN KEY (created_by_user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

-- --------------------------------------------------------------------
-- Create enum for user-account roles
-- --------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'role_enum') THEN
        CREATE TYPE role_enum AS ENUM ('user', 'admin');
    END IF;
END
$$;

-- --------------------------------------------------------------------
-- Create user_accounts table (join between users and accounts)
-- --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    account_id UUID NOT NULL,
    role role_enum NOT NULL DEFAULT 'user',

    CONSTRAINT fk_user_account_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,

    CONSTRAINT fk_user_account_account FOREIGN KEY (account_id)
        REFERENCES accounts (id) ON DELETE CASCADE,

    CONSTRAINT user_account_unique UNIQUE (user_id, account_id)
);

-- Optional: create explicit index (redundant with UNIQUE constraint)
CREATE INDEX IF NOT EXISTS idx_user_account_user
    ON user_accounts (user_id);
CREATE INDEX IF NOT EXISTS idx_user_account_account
    ON user_accounts (account_id);


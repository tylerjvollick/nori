-- =========================================
-- Create spaces table and add recent_spaces to users
-- =========================================

-- Create spaces table
CREATE TABLE IF NOT EXISTS spaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    account_id UUID NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_space_account FOREIGN KEY (account_id)
        REFERENCES accounts (id) ON DELETE CASCADE
);

-- Create index for faster lookups by account
CREATE INDEX IF NOT EXISTS idx_spaces_account_id
    ON spaces (account_id);

-- Create index for finding default spaces
CREATE INDEX IF NOT EXISTS idx_spaces_account_default
    ON spaces (account_id, is_default);

-- Add recent_spaces JSONB field to users table
ALTER TABLE users
    ADD COLUMN recent_spaces JSONB DEFAULT '[]'::jsonb;

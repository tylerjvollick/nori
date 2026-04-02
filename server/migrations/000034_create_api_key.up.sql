-- --------------------------------------------------------------------
-- Create api_keys table: API keys for programmatic access to accounts
-- --------------------------------------------------------------------

CREATE TABLE api_keys (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- Index on account_id for finding all keys for an account
CREATE INDEX idx_api_keys_account_id ON api_keys(account_id);

-- Index on key_hash for quick lookup during authentication
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- Index on is_active for filtering active keys
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);

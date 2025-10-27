-- Enable the uuid-ossp extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum type for GlobalRole
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'global_role') THEN
        CREATE TYPE global_role AS ENUM ('ADMIN', 'USER', 'VIEWER');
    END IF;
END$$;

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    global_role global_role NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT NULL,
    first_name TEXT NULL,
    last_name TEXT NULL,
    default_account_id UUID NULL
);

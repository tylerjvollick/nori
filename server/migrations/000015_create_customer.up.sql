-- Create customer table
CREATE TABLE customer (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(100),
    address TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on space_id for space-scoped queries
CREATE INDEX idx_customer_space_id ON customer(space_id);

-- Index on space_id + name for searching customers within a space
CREATE INDEX idx_customer_space_id_name ON customer(space_id, name);

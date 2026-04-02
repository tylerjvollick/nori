-- --------------------------------------------------------------------
-- Create space_member table: many-to-many join between users and spaces
-- --------------------------------------------------------------------

CREATE TABLE space_member (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Unique constraint: a user can only be a member of a space once
    CONSTRAINT uq_space_member_user_space UNIQUE (user_id, space_id)
);

-- Index on user_id for finding all spaces a user belongs to
CREATE INDEX idx_space_member_user_id ON space_member(user_id);

-- Index on space_id for finding all members of a space
CREATE INDEX idx_space_member_space_id ON space_member(space_id);

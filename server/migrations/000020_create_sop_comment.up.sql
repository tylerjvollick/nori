-- Create sop_comment table for comments and suggested edits on SOPs
CREATE TABLE sop_comment (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    sop_template_id INT NOT NULL REFERENCES sop_template(id) ON DELETE CASCADE,
    sop_step_id INT REFERENCES sop_step(id) ON DELETE CASCADE,
    sop_sub_step_id INT REFERENCES sop_sub_step(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    is_suggestion BOOLEAN NOT NULL DEFAULT false,
    is_resolved BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on sop_template_id for SOP-scoped comment queries (primary access pattern)
CREATE INDEX idx_sop_comment_sop_template_id ON sop_comment(sop_template_id);

-- Index on sop_step_id for step-scoped comment queries
CREATE INDEX idx_sop_comment_sop_step_id ON sop_comment(sop_step_id);

-- Index on sop_sub_step_id for sub-step-scoped comment queries
CREATE INDEX idx_sop_comment_sop_sub_step_id ON sop_comment(sop_sub_step_id);

-- Index on user_id for user's comment history
CREATE INDEX idx_sop_comment_user_id ON sop_comment(user_id);

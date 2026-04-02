-- Create sop_sub_step table for granular documentation within SOP steps.
-- Sub-steps are not individually timed; they serve as checklist items
-- for detailed procedure documentation (e.g., "Create Bridle Joint" step
-- has sub-steps: mark reference lines, set up jig, sneak up on cut, etc.).

CREATE TABLE IF NOT EXISTS sop_sub_step (
    id            SERIAL PRIMARY KEY,
    sop_step_id   INT NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    title         VARCHAR(255) NOT NULL,
    instructions  TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_sop_sub_step_sop_step
        FOREIGN KEY (sop_step_id)
        REFERENCES sop_step (id)
        ON DELETE CASCADE
);

-- Index for fetching sub-steps by parent step
CREATE INDEX IF NOT EXISTS idx_sop_sub_step_sop_step_id
    ON sop_sub_step (sop_step_id);

-- Composite index for ordered retrieval within a step
CREATE INDEX IF NOT EXISTS idx_sop_sub_step_step_order
    ON sop_sub_step (sop_step_id, display_order);

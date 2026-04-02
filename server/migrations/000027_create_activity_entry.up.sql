-- Create activity_entry table for chronological ticket activity logging.
-- Records step transitions, interruptions, comments, status changes, and more.
-- This is the "activity tab" on a ticket.

CREATE TYPE activity_entry_type AS ENUM (
    'status_change',
    'step_started',
    'step_completed',
    'step_paused',
    'step_resumed',
    'step_skipped',
    'comment',
    'interruption',
    'assignment_change',
    'link_added',
    'sop_edited',
    'cost_logged',
    'ticket_created'
);

CREATE TABLE activity_entry (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES ticket(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    entry_type activity_entry_type NOT NULL,
    description TEXT NOT NULL,
    linked_ticket_id UUID REFERENCES ticket(id) ON DELETE SET NULL,
    ticket_step_id UUID REFERENCES ticket_step(id) ON DELETE SET NULL,
    duration_seconds INT,
    old_value TEXT,
    new_value TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX idx_activity_entry_ticket ON activity_entry(ticket_id);
CREATE INDEX idx_activity_entry_ticket_chronological ON activity_entry(ticket_id, created_at ASC);
CREATE INDEX idx_activity_entry_ticket_step ON activity_entry(ticket_step_id) WHERE ticket_step_id IS NOT NULL;
CREATE INDEX idx_activity_entry_user ON activity_entry(user_id);
CREATE INDEX idx_activity_entry_type ON activity_entry(entry_type);

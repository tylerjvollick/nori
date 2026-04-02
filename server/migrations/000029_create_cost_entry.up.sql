-- Create cost_entry table for per-ticket cost tracking.
-- Supports labor, material, consumable, marketing, and other cost types.
-- Labor costs can be auto-computed from TimeEvents × labor rate or entered manually.

CREATE TYPE cost_type AS ENUM (
    'labor',
    'material',
    'consumable',
    'marketing',
    'other'
);

CREATE TABLE cost_entry (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES ticket(id) ON DELETE CASCADE,
    cost_type cost_type NOT NULL,
    description TEXT NOT NULL,
    amount NUMERIC(12,4) NOT NULL,
    quantity NUMERIC(12,4),
    unit VARCHAR(50),
    unit_cost NUMERIC(12,4),
    material_id UUID REFERENCES material(id) ON DELETE SET NULL,
    time_event_id UUID REFERENCES time_event(id) ON DELETE SET NULL,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX idx_cost_entry_ticket ON cost_entry(ticket_id);
CREATE INDEX idx_cost_entry_cost_type ON cost_entry(ticket_id, cost_type);
CREATE INDEX idx_cost_entry_material ON cost_entry(material_id) WHERE material_id IS NOT NULL;
CREATE INDEX idx_cost_entry_time_event ON cost_entry(time_event_id) WHERE time_event_id IS NOT NULL;
CREATE INDEX idx_cost_entry_created_by ON cost_entry(created_by_id);

-- Create time_event table for append-only, source-agnostic time tracking.
-- Every time-relevant action creates a TimeEvent. Records are immutable;
-- corrections are new events with backdated timestamps.
-- FK to task is added in 000038 after the task table exists.

CREATE TYPE time_event_type AS ENUM (
    'check_in',
    'check_out',
    'pause',
    'resume'
);

CREATE TYPE time_event_source AS ENUM (
    'manual',
    'web',
    'cli',
    'tap',
    'sensor',
    'api',
    'system'
);

CREATE TABLE time_event (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id),
    user_id UUID NOT NULL REFERENCES users(id),
    task_id VARCHAR(255),
    station_id UUID REFERENCES station(id) ON DELETE SET NULL,
    event_type time_event_type NOT NULL,
    source time_event_source NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX idx_time_event_space ON time_event(space_id);
CREATE INDEX idx_time_event_user ON time_event(user_id);
CREATE INDEX idx_time_event_task ON time_event(task_id) WHERE task_id IS NOT NULL;
CREATE INDEX idx_time_event_station ON time_event(station_id) WHERE station_id IS NOT NULL;
CREATE INDEX idx_time_event_space_timestamp ON time_event(space_id, "timestamp" DESC);
CREATE INDEX idx_time_event_user_timestamp ON time_event(user_id, "timestamp" DESC);

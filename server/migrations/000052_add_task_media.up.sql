CREATE TABLE task_media (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id        VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    file_path      TEXT NOT NULL,
    file_name      TEXT NOT NULL,
    mime_type      TEXT NOT NULL,
    file_size      BIGINT NOT NULL DEFAULT 0,
    duration       INT,
    display_order  INT NOT NULL DEFAULT 0,
    captured_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_media_task_id ON task_media(task_id);

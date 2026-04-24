CREATE TABLE sub_tasks (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id       VARCHAR(255) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT,
    display_order INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sub_tasks_task_id ON sub_tasks(task_id);

CREATE TABLE sub_task_images (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sub_task_id   UUID NOT NULL REFERENCES sub_tasks(id) ON DELETE CASCADE,
    image_url     TEXT NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sub_task_images_sub_task_id ON sub_task_images(sub_task_id);

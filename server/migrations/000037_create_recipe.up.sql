-- Create recipe_version_status enum
DO $$ BEGIN
    CREATE TYPE recipe_version_status AS ENUM ('draft', 'published', 'archived');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Create recipe table
CREATE TABLE recipe (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    category_id UUID,
    current_version_id INT,
    extends_recipe_id UUID REFERENCES recipe(id) ON DELETE SET NULL,
    created_by_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Unique index on (space_id, slug) for recipe uniqueness within a space
CREATE UNIQUE INDEX idx_recipe_space_id_slug ON recipe(space_id, slug) WHERE deleted_at IS NULL;

-- Index on space_id for space-scoped queries
CREATE INDEX idx_recipe_space_id ON recipe(space_id);

-- Index on created_by_id for author lookups
CREATE INDEX idx_recipe_created_by_id ON recipe(created_by_id);

-- Index on extends_recipe_id for inheritance lookups
CREATE INDEX idx_recipe_extends_recipe_id ON recipe(extends_recipe_id);

-- Index on deleted_at for soft delete filtering
CREATE INDEX idx_recipe_deleted_at ON recipe(deleted_at);

-- Create recipe_version table
CREATE TABLE recipe_version (
    id SERIAL PRIMARY KEY,
    recipe_id UUID NOT NULL REFERENCES recipe(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    status recipe_version_status NOT NULL DEFAULT 'draft',
    content TEXT NOT NULL,
    change_summary TEXT,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on recipe_id for version listing
CREATE INDEX idx_recipe_version_recipe_id ON recipe_version(recipe_id);

-- Unique index on (recipe_id, version_number) for version uniqueness
CREATE UNIQUE INDEX idx_recipe_version_recipe_id_version_number ON recipe_version(recipe_id, version_number);

-- Index on status for filtering by lifecycle state
CREATE INDEX idx_recipe_version_status ON recipe_version(status);

-- Index on author_id for author lookups
CREATE INDEX idx_recipe_version_author_id ON recipe_version(author_id);

-- Add foreign key from recipe.current_version_id to recipe_version.id
-- (deferred because recipe_version table must exist first)
ALTER TABLE recipe
    ADD CONSTRAINT fk_recipe_current_version
    FOREIGN KEY (current_version_id) REFERENCES recipe_version(id) ON DELETE SET NULL;

-- Add foreign keys from task table to recipe tables
-- (recipe_id and recipe_version_id were created without FKs in the task migration)
ALTER TABLE task
    ADD CONSTRAINT fk_task_recipe
    FOREIGN KEY (recipe_id) REFERENCES recipe(id) ON DELETE SET NULL;

ALTER TABLE task
    ADD CONSTRAINT fk_task_recipe_version
    FOREIGN KEY (recipe_version_id) REFERENCES recipe_version(id) ON DELETE SET NULL;

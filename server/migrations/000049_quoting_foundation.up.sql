-- Quoting foundation: material catalog fields, task materials, products,
-- variants, quotes, quote lines, and station labor rates (nori-xz5.1).
-- Note: spaces.default_labor_rate was already added in 000030.

-- Material catalog fields. The material table (000016) predates the quoting
-- system; the catalog adds supplier/sku and soft delete on top of it.
ALTER TABLE material ADD COLUMN supplier VARCHAR(255);
ALTER TABLE material ADD COLUMN sku VARCHAR(255);
ALTER TABLE material ADD COLUMN deleted_at TIMESTAMPTZ;

CREATE INDEX idx_material_deleted_at ON material(deleted_at);

-- Station hourly labor rate. Labor cost = estimated time x station rate,
-- falling back to spaces.default_labor_rate when null.
ALTER TABLE station ADD COLUMN costs_hour NUMERIC(12,4);

-- Materials attached to recipe/job tasks, scaled by batch quantity during
-- cost computation.
CREATE TABLE task_material (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id VARCHAR(255) NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES material(id) ON DELETE CASCADE,
    quantity_per_unit NUMERIC(12,4) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_task_material_task_id_material_id UNIQUE (task_id, material_id)
);

CREATE INDEX idx_task_material_task_id ON task_material(task_id);
CREATE INDEX idx_task_material_material_id ON task_material(material_id);

-- Product: what the shop sells.
CREATE TABLE product (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_product_space_id ON product(space_id);
CREATE INDEX idx_product_deleted_at ON product(deleted_at);

-- Product variant: a sellable configuration of a product, optionally bound
-- to a recipe with variable bindings and a customer-facing price.
CREATE TABLE product_variant (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    recipe_id UUID REFERENCES recipe(id) ON DELETE SET NULL,
    recipe_variables JSONB,
    price NUMERIC(12,4),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_product_variant_product_id ON product_variant(product_id);
CREATE INDEX idx_product_variant_recipe_id ON product_variant(recipe_id);

-- Quote: lightweight record with customer and line items. No task tree is
-- created until the quote is accepted and rolled into a job.
CREATE TABLE quote (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customer(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    notes TEXT,
    markup NUMERIC(12,4),
    override_total NUMERIC(12,4),
    accepted_at TIMESTAMPTZ,
    created_by_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_quote_status
        CHECK (status IN ('draft', 'sent', 'accepted', 'declined', 'cancelled'))
);

CREATE INDEX idx_quote_space_id ON quote(space_id);
CREATE INDEX idx_quote_customer_id ON quote(customer_id);
CREATE INDEX idx_quote_status ON quote(status);

-- Quote line: a product variant (or bare recipe for non-product quotes) and
-- quantity, with the dry-run cost snapshot frozen at quote time.
CREATE TABLE quote_line (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    quote_id UUID NOT NULL REFERENCES quote(id) ON DELETE CASCADE,
    product_variant_id UUID REFERENCES product_variant(id) ON DELETE SET NULL,
    recipe_id UUID REFERENCES recipe(id) ON DELETE SET NULL,
    quantity INT NOT NULL,
    unit_price NUMERIC(12,4),
    cost_snapshot JSONB,
    recipe_variables JSONB,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_quote_line_quote_id ON quote_line(quote_id);
CREATE INDEX idx_quote_line_product_variant_id ON quote_line(product_variant_id);

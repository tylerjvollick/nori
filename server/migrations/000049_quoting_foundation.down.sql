-- Reverse quoting foundation (nori-xz5.1).

DROP TABLE IF EXISTS quote_line;
DROP TABLE IF EXISTS quote;
DROP TABLE IF EXISTS product_variant;
DROP TABLE IF EXISTS product;
DROP TABLE IF EXISTS task_material;

ALTER TABLE station DROP COLUMN IF EXISTS costs_hour;

DROP INDEX IF EXISTS idx_material_deleted_at;
ALTER TABLE material DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE material DROP COLUMN IF EXISTS sku;
ALTER TABLE material DROP COLUMN IF EXISTS supplier;

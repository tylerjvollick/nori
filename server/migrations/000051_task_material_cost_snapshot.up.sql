-- Snapshot of the material's unit cost at the time a recipe is rolled into a
-- job. Null on recipe task_materials; set when cloned onto job tasks so the
-- job's costs are frozen even if the catalog price later changes (nori-xz5.3).
ALTER TABLE task_material ADD COLUMN snapshotted_unit_cost NUMERIC(12,4);

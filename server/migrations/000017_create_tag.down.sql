-- Remove tag table
DROP INDEX IF EXISTS idx_tag_space_id;
DROP INDEX IF EXISTS idx_tag_space_id_name;
DROP TABLE IF EXISTS tag;

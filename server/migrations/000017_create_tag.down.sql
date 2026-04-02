-- Remove indexes and join tables first (depend on tag)
DROP INDEX IF EXISTS idx_ticket_tag_tag_id;
DROP INDEX IF EXISTS idx_ticket_tag_ticket_id;
DROP TABLE IF EXISTS ticket_tag;

DROP INDEX IF EXISTS idx_sop_template_tag_tag_id;
DROP INDEX IF EXISTS idx_sop_template_tag_template_id;
DROP TABLE IF EXISTS sop_template_tag;

-- Remove tag table
DROP INDEX IF EXISTS idx_tag_space_id;
DROP INDEX IF EXISTS idx_tag_space_id_name;
DROP TABLE IF EXISTS tag;

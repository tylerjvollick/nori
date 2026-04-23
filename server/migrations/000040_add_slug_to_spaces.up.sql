-- Add slug column to spaces table (2-5 chars uppercase, like Jira project keys)
ALTER TABLE spaces ADD COLUMN slug VARCHAR(10);

-- Backfill existing spaces with auto-generated slugs from name
-- Takes first letter of each word (up to 3), uppercase. Falls back to first 3 chars.
UPDATE spaces
SET slug = UPPER(
    CASE
        WHEN array_length(string_to_array(regexp_replace(name, '[^a-zA-Z ]', '', 'g'), ' '), 1) >= 2 THEN
            LEFT(
                array_to_string(
                    ARRAY(
                        SELECT LEFT(word, 1)
                        FROM unnest(string_to_array(regexp_replace(name, '[^a-zA-Z ]', '', 'g'), ' ')) AS word
                        WHERE word != ''
                        LIMIT 3
                    ),
                    ''
                ),
                5
            )
        ELSE
            LEFT(regexp_replace(name, '[^a-zA-Z]', '', 'g'), 3)
    END
);

-- Handle any empty slugs (e.g., names with no alpha chars) by using 'SPC' + id suffix
UPDATE spaces
SET slug = 'SPC'
WHERE slug IS NULL OR slug = '';

-- Handle duplicate slugs within the same account by appending a numeric suffix
DO $$
DECLARE
    r RECORD;
    counter INT;
    new_slug TEXT;
BEGIN
    FOR r IN
        SELECT id, slug, account_id,
               ROW_NUMBER() OVER (PARTITION BY account_id, slug ORDER BY created_at) AS rn
        FROM spaces
    LOOP
        IF r.rn > 1 THEN
            counter := r.rn;
            new_slug := LEFT(r.slug, 4) || counter::TEXT;
            UPDATE spaces SET slug = new_slug WHERE id = r.id;
        END IF;
    END LOOP;
END $$;

-- Now make slug NOT NULL and add unique constraint per account
ALTER TABLE spaces ALTER COLUMN slug SET NOT NULL;
CREATE UNIQUE INDEX idx_spaces_account_slug ON spaces (account_id, slug);

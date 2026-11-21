ALTER TABLE ticket_comments
DROP CONSTRAINT IF EXISTS ticket_comments_author_type_check;

ALTER TABLE ticket_comments
DROP COLUMN IF EXISTS author_type;
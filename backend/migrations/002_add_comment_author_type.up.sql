ALTER TABLE ticket_comments
ADD COLUMN author_type VARCHAR(20);

ALTER TABLE ticket_comments
ADD CONSTRAINT ticket_comments_author_type_check
CHECK (
    author_type IS NULL
    OR author_type IN ('customer', 'agent', 'system')
);
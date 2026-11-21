DROP INDEX IF EXISTS idx_tickets_deleted_at;

ALTER TABLE tickets DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE tickets ADD COLUMN deleted_at TIMESTAMPTZ NULL;

CREATE INDEX idx_tickets_deleted_at ON tickets (deleted_at) WHERE deleted_at IS NULL;

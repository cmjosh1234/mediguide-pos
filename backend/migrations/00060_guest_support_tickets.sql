-- +goose Up

ALTER TABLE support_tickets ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE support_tickets ADD COLUMN requester_name text;
ALTER TABLE support_tickets ADD COLUMN requester_email text;
ALTER TABLE support_tickets
  ADD CONSTRAINT chk_support_tickets_requester
  CHECK (user_id IS NOT NULL OR (requester_name IS NOT NULL AND requester_email IS NOT NULL));
CREATE INDEX idx_support_tickets_requester_email ON support_tickets(requester_email);

-- +goose Down

DROP INDEX IF EXISTS idx_support_tickets_requester_email;
ALTER TABLE support_tickets DROP CONSTRAINT IF EXISTS chk_support_tickets_requester;
DELETE FROM support_ticket_replies WHERE ticket_id IN (SELECT id FROM support_tickets WHERE user_id IS NULL);
DELETE FROM support_tickets WHERE user_id IS NULL;
ALTER TABLE support_tickets DROP COLUMN requester_email;
ALTER TABLE support_tickets DROP COLUMN requester_name;
ALTER TABLE support_tickets ALTER COLUMN user_id SET NOT NULL;

-- +goose Up
ALTER TABLE users ADD COLUMN email_verified boolean NOT NULL DEFAULT false;
-- Preserve proven historical email ownership separately from administrative approval.
UPDATE users SET email_verified = true WHERE EXISTS (
 SELECT 1 FROM audit_logs WHERE action = 'user.email_verified' AND entity_id = users.id::text
);
-- Keep existing administrative approval: expired audit history cannot prove its absence.
CREATE TABLE account_email_deliveries (
 id uuid PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id),
 token_id uuid NOT NULL,
 purpose text NOT NULL,
 encrypted_token text NOT NULL,
 status text NOT NULL,
 attempts integer NOT NULL DEFAULT 0,
 next_attempt_at timestamptz NOT NULL,
 lease_until timestamptz,
 sent_at timestamptz,
 last_error_code text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX account_email_delivery_due ON account_email_deliveries(status, next_attempt_at);
CREATE INDEX account_email_delivery_user ON account_email_deliveries(user_id);
CREATE INDEX account_email_delivery_token ON account_email_deliveries(token_id);

-- +goose Down
DROP TABLE account_email_deliveries;
UPDATE users SET verified = verified OR email_verified;
ALTER TABLE users DROP COLUMN email_verified;

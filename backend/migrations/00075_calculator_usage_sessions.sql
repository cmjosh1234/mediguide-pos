-- +goose Up
ALTER TABLE calculator_usage_logs ADD COLUMN idempotency_key varchar(128);
CREATE UNIQUE INDEX calculator_usage_user_key ON calculator_usage_logs (user_id, idempotency_key);

-- +goose Down
DROP INDEX calculator_usage_user_key;
ALTER TABLE calculator_usage_logs DROP COLUMN idempotency_key;

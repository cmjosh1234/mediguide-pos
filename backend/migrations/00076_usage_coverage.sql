-- +goose Up
ALTER TABLE drug_usage_logs ADD COLUMN idempotency_key varchar(128);
ALTER TABLE facility_usage_logs ADD COLUMN idempotency_key varchar(128);
CREATE UNIQUE INDEX drug_usage_user_key ON drug_usage_logs(user_id, idempotency_key);
CREATE UNIQUE INDEX facility_usage_user_key ON facility_usage_logs(user_id, idempotency_key);
CREATE TABLE medical_guideline_usage_logs (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id),
 medical_guideline_id uuid NOT NULL REFERENCES medical_guidelines(id),
 idempotency_key varchar(128), created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz
);
CREATE UNIQUE INDEX medical_guideline_usage_user_key ON medical_guideline_usage_logs(user_id,idempotency_key);
CREATE INDEX medical_guideline_usage_created ON medical_guideline_usage_logs(created_at);
CREATE TABLE feature_usage_logs (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id),
 feature varchar(64) NOT NULL, idempotency_key varchar(128),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz
);
CREATE UNIQUE INDEX feature_usage_user_key ON feature_usage_logs(user_id,idempotency_key);
CREATE INDEX feature_usage_created ON feature_usage_logs(created_at);
-- +goose Down
DROP TABLE feature_usage_logs;
DROP TABLE medical_guideline_usage_logs;
DROP INDEX facility_usage_user_key;
DROP INDEX drug_usage_user_key;
ALTER TABLE facility_usage_logs DROP COLUMN idempotency_key;
ALTER TABLE drug_usage_logs DROP COLUMN idempotency_key;

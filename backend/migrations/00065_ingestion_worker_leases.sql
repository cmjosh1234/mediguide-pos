-- +goose Up
ALTER TABLE ingestion_jobs
  ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_expired_lease
  ON ingestion_jobs (lease_expires_at)
  WHERE status = 'running' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_retry_ready
  ON ingestion_jobs (next_attempt_at, created_at)
  WHERE status = 'failed' AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_ingestion_jobs_retry_ready;
DROP INDEX IF EXISTS idx_ingestion_jobs_expired_lease;
ALTER TABLE ingestion_jobs
  DROP COLUMN IF EXISTS next_attempt_at,
  DROP COLUMN IF EXISTS lease_expires_at,
  DROP COLUMN IF EXISTS heartbeat_at;

-- +goose Up
ALTER TABLE ingestion_jobs
  ADD COLUMN IF NOT EXISTS worker_id TEXT,
  ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_queue_claim
  ON ingestion_jobs (status, created_at, id)
  WHERE deleted_at IS NULL
    AND job_type IN ('pdf_ingestion', 'markdown_ingestion', 'original_text_index');

-- +goose Down
DROP INDEX IF EXISTS idx_ingestion_jobs_queue_claim;
ALTER TABLE ingestion_jobs
  DROP COLUMN IF EXISTS claimed_at,
  DROP COLUMN IF EXISTS worker_id;

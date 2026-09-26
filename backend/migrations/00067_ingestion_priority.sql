-- +goose Up
ALTER TABLE ingestion_jobs
  ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 50
  CHECK (priority BETWEEN 0 AND 100);

CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_priority_queue
  ON ingestion_jobs (status, priority DESC, created_at ASC, id ASC)
  WHERE deleted_at IS NULL
    AND job_type IN ('pdf_ingestion', 'markdown_ingestion', 'original_text_index');

-- +goose Down
DROP INDEX IF EXISTS idx_ingestion_jobs_priority_queue;
ALTER TABLE ingestion_jobs DROP COLUMN IF EXISTS priority;

-- +goose Up

CREATE TABLE notification_topic_jobs (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  topic text NOT NULL,
  source_type text NOT NULL,
  source_id uuid NOT NULL,
  status text NOT NULL DEFAULT 'pending',
  idempotency_key text NOT NULL,
  payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  attempt_count integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 8,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  locked_at timestamptz,
  locked_by text,
  provider_message_id text,
  last_error_code text,
  last_error_message text,
  completed_at timestamptz,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT chk_notification_topic_jobs_topic CHECK (topic LIKE 'public-%'),
  CONSTRAINT chk_notification_topic_jobs_status CHECK (status IN ('pending', 'processing', 'retry', 'accepted', 'failed')),
  CONSTRAINT chk_notification_topic_jobs_attempts CHECK (attempt_count >= 0 AND max_attempts BETWEEN 1 AND 20)
);
CREATE UNIQUE INDEX idx_notification_topic_jobs_idempotency
  ON notification_topic_jobs(idempotency_key) WHERE deleted_at IS NULL;
CREATE INDEX idx_notification_topic_jobs_claim
  ON notification_topic_jobs(status, next_attempt_at, created_at)
  WHERE deleted_at IS NULL AND status IN ('pending', 'retry');

-- +goose Down

DROP TABLE IF EXISTS notification_topic_jobs;

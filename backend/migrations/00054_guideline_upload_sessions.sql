-- +goose Up
CREATE TABLE guideline_uploads (
 id uuid PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 version_id uuid NOT NULL REFERENCES guideline_versions(id),
 user_id uuid NOT NULL REFERENCES users(id), filename text NOT NULL,
 size_bytes bigint NOT NULL CHECK(size_bytes > 0), checksum text NOT NULL,
 object_key text NOT NULL, multipart_id text NOT NULL,
 part_size bigint NOT NULL, status text NOT NULL, notified_status text NOT NULL DEFAULT '',
 job_id uuid REFERENCES ingestion_jobs(id), expires_at timestamptz NOT NULL
);
CREATE INDEX guideline_upload_owner ON guideline_uploads(user_id,version_id,created_at DESC);
CREATE INDEX guideline_upload_expiry ON guideline_uploads(expires_at) WHERE status = 'uploading';
ALTER TABLE ingestion_jobs ADD COLUMN metrics_json jsonb NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE ingestion_jobs DROP COLUMN metrics_json;
DROP TABLE guideline_uploads;

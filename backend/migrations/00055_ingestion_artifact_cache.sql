-- +goose Up
-- Internal computed artifacts only. No publication/review state or ownership.
CREATE TABLE ingestion_artifact_cache (
 cache_key text PRIMARY KEY,
 kind text NOT NULL CHECK (kind IN ('extraction','ocr','embedding')),
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ingestion_artifact_cache_kind_created ON ingestion_artifact_cache(kind, created_at);

-- +goose Down
DROP TABLE ingestion_artifact_cache;

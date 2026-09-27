-- +goose Up
-- Link kinds publish an external https URL instead of an uploaded file or
-- extracted content. A kind publishes either as uploaded or as a link, not both.
ALTER TABLE document_kinds ADD COLUMN publish_as_link boolean NOT NULL DEFAULT false;
ALTER TABLE document_kinds ADD CONSTRAINT document_kinds_publish_mode_check
  CHECK (NOT (publish_as_uploaded AND publish_as_link));
UPDATE document_kinds SET publish_as_link = true, publish_as_uploaded = false WHERE slug = 'link';

-- The external URL a link version points readers to.
ALTER TABLE guideline_versions ADD COLUMN external_url text NOT NULL DEFAULT '';
ALTER TABLE guideline_versions ADD CONSTRAINT guideline_versions_external_url_check
  CHECK (length(external_url) <= 2048);

-- +goose Down
ALTER TABLE guideline_versions DROP CONSTRAINT IF EXISTS guideline_versions_external_url_check;
ALTER TABLE guideline_versions DROP COLUMN IF EXISTS external_url;
ALTER TABLE document_kinds DROP CONSTRAINT IF EXISTS document_kinds_publish_mode_check;
ALTER TABLE document_kinds DROP COLUMN IF EXISTS publish_as_link;

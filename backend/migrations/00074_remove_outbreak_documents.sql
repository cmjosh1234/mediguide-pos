-- +goose Up
-- Outbreak documents and SOPs (uploaded files owned by an outbreak) have been
-- removed from every client and the API. Typed resources stay in
-- outbreak_resources; only the document rows and their document-only columns go.
UPDATE notifications
SET action_json = '{"type":"none","parameters":{}}'::jsonb
WHERE action_json->>'type' = 'outbreak_document';

DELETE FROM content_pillar_items WHERE content_type IN ('outbreak_document', 'form');
DELETE FROM content_disease_assignments WHERE content_type IN ('outbreak_document', 'form');
DELETE FROM audit_logs WHERE entity_type = 'outbreak_document';
DELETE FROM notification_templates WHERE template_key LIKE 'outbreak-document-%';

UPDATE outbreak_resources SET supersedes_id = NULL
WHERE supersedes_id IN (
  SELECT id FROM outbreak_resources WHERE resource_type IN ('managed_document', 'downloadable_asset')
);
DELETE FROM outbreak_resources WHERE resource_type IN ('managed_document', 'downloadable_asset');

ALTER TABLE content_disease_assignments DROP CONSTRAINT content_disease_assignments_content_type_check;
ALTER TABLE content_disease_assignments ADD CONSTRAINT content_disease_assignments_content_type_check
  CHECK (content_type IN ('guideline','outbreak','situation_report','algorithm','clinical_tool','drug_reference'));
ALTER TABLE content_pillar_items DROP CONSTRAINT content_pillar_items_content_type_check;
ALTER TABLE content_pillar_items ADD CONSTRAINT content_pillar_items_content_type_check
  CHECK (content_type IN ('guideline','situation_report','algorithm','clinical_tool','drug_reference','internal_route','approved_external_url'));

DROP INDEX IF EXISTS idx_outbreak_resources_documents;
DROP INDEX IF EXISTS idx_outbreak_resources_document_authority;
DROP INDEX IF EXISTS idx_outbreak_resources_document_search;
DROP INDEX IF EXISTS idx_outbreak_resources_document_search_weighted;
DROP INDEX IF EXISTS idx_outbreak_resources_document_title_trgm;
DROP INDEX IF EXISTS idx_outbreak_resources_document_number_trgm;
DROP INDEX IF EXISTS idx_outbreak_resources_published_document_version;
DROP INDEX IF EXISTS idx_outbreak_resources_storage_key;

ALTER TABLE outbreak_resources
  DROP CONSTRAINT IF EXISTS outbreak_resources_checksum_check,
  DROP CONSTRAINT IF EXISTS outbreak_resources_document_dates_check,
  DROP CONSTRAINT IF EXISTS outbreak_resources_page_count_check,
  DROP CONSTRAINT IF EXISTS outbreak_resources_file_size_check,
  DROP COLUMN document_number,
  DROP COLUMN version,
  DROP COLUMN language,
  DROP COLUMN audience,
  DROP COLUMN effective_date,
  DROP COLUMN review_date,
  DROP COLUMN expires_at,
  DROP COLUMN storage_key,
  DROP COLUMN original_filename,
  DROP COLUMN mime_type,
  DROP COLUMN file_size,
  DROP COLUMN checksum_sha256,
  DROP COLUMN page_count,
  DROP COLUMN search_content,
  DROP COLUMN rendered_content,
  DROP COLUMN content_format,
  DROP COLUMN extraction_status,
  DROP COLUMN extracted_at,
  DROP COLUMN search_headings,
  DROP COLUMN extraction_error,
  DROP COLUMN extraction_source_checksum,
  DROP COLUMN derived_content_checksum,
  DROP COLUMN search_index_status,
  DROP COLUMN search_schema_version,
  DROP COLUMN content_sections,
  DROP COLUMN source_page_map,
  DROP COLUMN indexed_at;

-- +goose Down
-- Recreates the empty document schema only; deleted documents, their files,
-- assignments, hub items, audit rows and notification templates are not restored.
ALTER TABLE outbreak_resources
  ADD COLUMN document_number TEXT NOT NULL DEFAULT '',
  ADD COLUMN version TEXT NOT NULL DEFAULT '',
  ADD COLUMN language TEXT NOT NULL DEFAULT 'en',
  ADD COLUMN audience TEXT NOT NULL DEFAULT '',
  ADD COLUMN effective_date TIMESTAMPTZ,
  ADD COLUMN review_date TIMESTAMPTZ,
  ADD COLUMN expires_at TIMESTAMPTZ,
  ADD COLUMN storage_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN original_filename TEXT NOT NULL DEFAULT '',
  ADD COLUMN mime_type TEXT NOT NULL DEFAULT '',
  ADD COLUMN file_size BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN checksum_sha256 TEXT NOT NULL DEFAULT '',
  ADD COLUMN page_count INTEGER,
  ADD COLUMN search_content text NOT NULL DEFAULT '',
  ADD COLUMN rendered_content text NOT NULL DEFAULT '',
  ADD COLUMN content_format varchar(32) NOT NULL DEFAULT '',
  ADD COLUMN extraction_status varchar(32) NOT NULL DEFAULT 'not_available',
  ADD COLUMN extracted_at timestamptz,
  ADD COLUMN search_headings text NOT NULL DEFAULT '',
  ADD COLUMN extraction_error text NOT NULL DEFAULT '',
  ADD COLUMN extraction_source_checksum varchar(64) NOT NULL DEFAULT '',
  ADD COLUMN derived_content_checksum varchar(64) NOT NULL DEFAULT '',
  ADD COLUMN search_index_status varchar(32) NOT NULL DEFAULT 'not_indexed',
  ADD COLUMN search_schema_version integer NOT NULL DEFAULT 1,
  ADD COLUMN content_sections jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN source_page_map jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN indexed_at timestamptz,
  ADD CONSTRAINT outbreak_resources_file_size_check CHECK (file_size >= 0),
  ADD CONSTRAINT outbreak_resources_page_count_check CHECK (page_count IS NULL OR page_count > 0),
  ADD CONSTRAINT outbreak_resources_document_dates_check CHECK (
    (review_date IS NULL OR effective_date IS NULL OR review_date >= effective_date)
    AND (expires_at IS NULL OR effective_date IS NULL OR expires_at >= effective_date)
  ),
  ADD CONSTRAINT outbreak_resources_checksum_check CHECK (checksum_sha256 = '' OR checksum_sha256 ~ '^[0-9a-f]{64}$');

CREATE INDEX idx_outbreak_resources_documents
  ON outbreak_resources (outbreak_id, document_kind, status, effective_date DESC, id DESC)
  WHERE deleted_at IS NULL AND resource_type IN ('managed_document','downloadable_asset');
CREATE INDEX idx_outbreak_resources_document_authority
  ON outbreak_resources (lower(issuing_authority), language, id)
  WHERE deleted_at IS NULL AND resource_type IN ('managed_document','downloadable_asset');
CREATE INDEX idx_outbreak_resources_document_search
  ON outbreak_resources USING GIN (to_tsvector('simple', COALESCE(title, '') || ' ' || COALESCE(description, '') || ' ' || COALESCE(document_number, '') || ' ' || COALESCE(issuing_authority, '') || ' ' || COALESCE(document_kind, '') || ' ' || COALESCE(audience, '')))
  WHERE deleted_at IS NULL AND resource_type IN ('managed_document','downloadable_asset');
CREATE UNIQUE INDEX idx_outbreak_resources_published_document_version
  ON outbreak_resources (outbreak_id, lower(document_number), lower(version))
  WHERE deleted_at IS NULL AND status = 'published' AND resource_type IN ('managed_document','downloadable_asset') AND document_number <> '' AND version <> '';
CREATE INDEX idx_outbreak_resources_storage_key
  ON outbreak_resources (storage_key)
  WHERE deleted_at IS NULL AND storage_key <> '';
CREATE INDEX idx_outbreak_resources_document_title_trgm
  ON outbreak_resources USING GIN (lower(title) gin_trgm_ops)
  WHERE deleted_at IS NULL AND resource_type IN ('managed_document','downloadable_asset');
CREATE INDEX idx_outbreak_resources_document_number_trgm
  ON outbreak_resources USING GIN (lower(document_number) gin_trgm_ops)
  WHERE deleted_at IS NULL AND resource_type IN ('managed_document','downloadable_asset') AND document_number <> '';
CREATE INDEX idx_outbreak_resources_document_search_weighted
  ON outbreak_resources USING GIN ((
    setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(document_number, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(issuing_authority, '') || ' ' || COALESCE(document_kind, '') || ' ' || COALESCE(audience, '')), 'B') ||
    setweight(to_tsvector('simple', COALESCE(search_headings, '')), 'B') ||
    setweight(to_tsvector('simple', COALESCE(description, '')), 'C') ||
    setweight(to_tsvector('simple', COALESCE(search_content, '')), 'D')
  ))
  WHERE deleted_at IS NULL AND status = 'published' AND approved_at IS NOT NULL AND withdrawn_at IS NULL AND resource_type IN ('managed_document','downloadable_asset');

ALTER TABLE content_disease_assignments DROP CONSTRAINT content_disease_assignments_content_type_check;
ALTER TABLE content_disease_assignments ADD CONSTRAINT content_disease_assignments_content_type_check
  CHECK (content_type IN ('guideline','outbreak','outbreak_document','situation_report','algorithm','clinical_tool','form','drug_reference'));
ALTER TABLE content_pillar_items DROP CONSTRAINT content_pillar_items_content_type_check;
ALTER TABLE content_pillar_items ADD CONSTRAINT content_pillar_items_content_type_check
  CHECK (content_type IN ('guideline','outbreak_document','situation_report','algorithm','clinical_tool','form','drug_reference','internal_route','approved_external_url'));

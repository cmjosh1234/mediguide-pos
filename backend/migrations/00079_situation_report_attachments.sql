-- +goose Up
-- Situation reports attach published documents from the guideline library
-- instead of uploading their own PDF. Attachments are reviewed and published
-- with the report; earlier uploaded PDFs (situation_report_assets) stay served.
CREATE TABLE situation_report_attachments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  situation_report_id UUID NOT NULL REFERENCES situation_reports(id) ON DELETE CASCADE,
  title TEXT NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 240),
  description TEXT NOT NULL DEFAULT '',
  issuing_organization TEXT NOT NULL DEFAULT '',
  document_kind TEXT NOT NULL,
  url TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  lock_version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_situation_report_attachments_report ON situation_report_attachments(situation_report_id, sort_order ASC)
  WHERE deleted_at IS NULL;

-- Report attachments are kept as the uploaded file, like forms. No document
-- uses this kind yet, so nothing already published changes.
UPDATE document_kinds SET publish_as_uploaded = true WHERE slug = 'situation_report_attachment';

-- +goose Down
UPDATE document_kinds SET publish_as_uploaded = false WHERE slug = 'situation_report_attachment';
DROP TABLE IF EXISTS situation_report_attachments;

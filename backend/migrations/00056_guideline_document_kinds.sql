-- +goose Up
CREATE TABLE guideline_document_kinds (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
  slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  description text,
  sort_order integer NOT NULL DEFAULT 0 CHECK (sort_order BETWEEN 0 AND 10000),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE UNIQUE INDEX idx_guideline_document_kinds_slug
  ON guideline_document_kinds (slug)
  WHERE deleted_at IS NULL;

INSERT INTO guideline_document_kinds (id, name, slug, description, sort_order) VALUES
  ('94000000-0000-4000-8000-000000000001', 'Guideline', 'guideline', 'Clinical guidance publications.', 10),
  ('94000000-0000-4000-8000-000000000002', 'Form', 'form', 'Standard clinical and reporting forms.', 20);

-- Every document has exactly one kind. Existing documents are guidelines; the
-- default keeps direct inserts valid while the API always sets the kind.
ALTER TABLE guideline_documents
  ADD COLUMN document_kind_id uuid NOT NULL
    DEFAULT '94000000-0000-4000-8000-000000000001'
    REFERENCES guideline_document_kinds(id) ON DELETE RESTRICT;
CREATE INDEX idx_guideline_documents_document_kind
  ON guideline_documents (document_kind_id)
  WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_guideline_documents_document_kind;
ALTER TABLE guideline_documents DROP COLUMN IF EXISTS document_kind_id;
DROP TABLE IF EXISTS guideline_document_kinds;

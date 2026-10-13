-- +goose Up
INSERT INTO document_kinds (id, name, slug, description, sort_order) VALUES
  ('94000000-0000-4000-8000-000000000018', 'Manual', 'manual', 'Detailed reference and operational manuals.', 145),
  ('94000000-0000-4000-8000-000000000019', 'Framework', 'framework', 'Strategic, implementation and assessment frameworks.', 146)
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
-- Preserve documents when rolling back the default taxonomy.
UPDATE guideline_documents SET document_kind_id = (
  SELECT id FROM document_kinds WHERE slug = 'other'
) WHERE document_kind_id IN (
  SELECT id FROM document_kinds WHERE slug IN ('manual', 'framework')
);
DELETE FROM document_kinds WHERE slug IN ('manual', 'framework');

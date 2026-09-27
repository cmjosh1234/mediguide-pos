-- +goose Up
INSERT INTO document_kinds (id, name, slug, description, sort_order) VALUES
  ('94000000-0000-4000-8000-000000000017', 'Link', 'link', 'External links to resources hosted elsewhere.', 155)
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DELETE FROM document_kinds WHERE slug = 'link';

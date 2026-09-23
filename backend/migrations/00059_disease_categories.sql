-- +goose Up
-- Diseases share the guideline category vocabulary, so a single category can
-- filter both the diseases and the guidelines filed under it.
CREATE TABLE disease_categories (
  disease_id uuid NOT NULL REFERENCES diseases(id) ON DELETE CASCADE,
  category_id uuid NOT NULL REFERENCES guideline_categories(id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (disease_id, category_id)
);
CREATE INDEX idx_disease_categories_category
  ON disease_categories (category_id, disease_id);

-- +goose Down
DROP TABLE IF EXISTS disease_categories;

-- +goose Up
ALTER TABLE outbreaks ADD COLUMN disease_id uuid REFERENCES diseases(id) ON DELETE RESTRICT;

-- Backfill from the primary disease assignment already computed for outbreaks
-- by migration 00049 (content_disease_assignments, content_type='outbreak').
-- Outbreaks whose original disease_type had no unambiguous taxonomy match are
-- left with disease_id NULL; the original free text remains recoverable from
-- disease_taxonomy_migration_report.source_value.
UPDATE outbreaks o
SET disease_id = a.disease_id
FROM content_disease_assignments a
WHERE a.content_type = 'outbreak'
  AND a.content_id = o.id
  AND a.is_primary
  AND a.deleted_at IS NULL
  AND o.deleted_at IS NULL;

DROP INDEX IF EXISTS idx_outbreaks_public_disease;
DROP INDEX IF EXISTS idx_outbreaks_public_search;

CREATE INDEX idx_outbreaks_public_disease
  ON outbreaks (disease_id, last_update DESC, id DESC)
  WHERE deleted_at IS NULL AND withdrawn_at IS NULL AND published_at IS NOT NULL;
CREATE INDEX idx_outbreaks_public_search
  ON outbreaks USING GIN (
    to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(summary, '') || ' ' || coalesce(geographic_area, '') || ' ' || coalesce(source_organization, ''))
  )
  WHERE deleted_at IS NULL AND withdrawn_at IS NULL AND published_at IS NOT NULL;

ALTER TABLE outbreaks DROP COLUMN disease_type;

-- +goose Down
ALTER TABLE outbreaks ADD COLUMN disease_type TEXT NOT NULL DEFAULT '';
UPDATE outbreaks o
SET disease_type = d.name
FROM diseases d
WHERE d.id = o.disease_id;

DROP INDEX IF EXISTS idx_outbreaks_public_search;
DROP INDEX IF EXISTS idx_outbreaks_public_disease;

CREATE INDEX idx_outbreaks_public_disease
  ON outbreaks (lower(disease_type), last_update DESC, id DESC)
  WHERE deleted_at IS NULL AND withdrawn_at IS NULL AND published_at IS NOT NULL;
CREATE INDEX idx_outbreaks_public_search
  ON outbreaks USING GIN (
    to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(summary, '') || ' ' || coalesce(disease_type, '') || ' ' || coalesce(geographic_area, '') || ' ' || coalesce(source_organization, ''))
  )
  WHERE deleted_at IS NULL AND withdrawn_at IS NULL AND published_at IS NOT NULL;

ALTER TABLE outbreaks DROP COLUMN disease_id;

-- +goose Up
ALTER TABLE guideline_versions
  ADD COLUMN source_version_id uuid REFERENCES guideline_versions(id),
  ADD COLUMN revision_source text NOT NULL DEFAULT '',
  ADD COLUMN revision_warnings jsonb NOT NULL DEFAULT '[]';
CREATE INDEX guideline_versions_source_version_idx ON guideline_versions(source_version_id);

-- +goose Down
DROP INDEX guideline_versions_source_version_idx;
ALTER TABLE guideline_versions DROP COLUMN revision_warnings, DROP COLUMN revision_source, DROP COLUMN source_version_id;

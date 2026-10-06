-- +goose Up
-- Historical engagement survives retirement; no new events target these IDs.
ALTER TABLE medical_guideline_usage_logs
  DROP CONSTRAINT medical_guideline_usage_logs_medical_guideline_id_fkey;
ALTER TABLE medical_guideline_usage_logs RENAME TO historical_guideline_usage_logs;
ALTER TABLE historical_guideline_usage_logs RENAME COLUMN medical_guideline_id TO legacy_guideline_id;
ALTER INDEX medical_guideline_usage_user_key RENAME TO historical_guideline_usage_user_key;
ALTER INDEX medical_guideline_usage_created RENAME TO historical_guideline_usage_created;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_disease_taxonomy_migration_report()
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  DELETE FROM disease_taxonomy_migration_report;
  WITH legacy_values AS (
    -- Outbreaks already use disease_id (migration 61); only documents still
    -- have a free-text taxonomy field to reconcile.
    SELECT 'guideline_documents'::text source_table, id source_id,
      'program_area'::text source_field, trim(program_area) source_value
    FROM guideline_documents WHERE deleted_at IS NULL AND trim(coalesce(program_area, '')) <> ''
  ), normalized AS (
    SELECT *, trim(regexp_replace(lower(source_value), '[^[:alnum:]]+', ' ', 'g')) normalized_value
    FROM legacy_values
  ), matches AS (
    SELECT n.*, coalesce(array_agg(DISTINCT d.id ORDER BY d.id) FILTER (WHERE d.id IS NOT NULL), ARRAY[]::uuid[]) candidate_ids
    FROM normalized n
    LEFT JOIN diseases d ON d.deleted_at IS NULL AND d.status = 'active' AND (
      d.normalized_name = n.normalized_value OR EXISTS (
        SELECT 1 FROM disease_aliases da
        WHERE da.disease_id = d.id AND da.deleted_at IS NULL AND da.normalized_alias = n.normalized_value
      )
    )
    GROUP BY n.source_table, n.source_id, n.source_field, n.source_value, n.normalized_value
  )
  INSERT INTO disease_taxonomy_migration_report (
    source_table, source_id, source_field, source_value, normalized_value,
    resolution_status, disease_id, candidate_disease_ids
  )
  SELECT source_table, source_id, source_field, source_value, normalized_value,
    CASE cardinality(candidate_ids) WHEN 0 THEN 'unmatched' WHEN 1 THEN 'matched' ELSE 'ambiguous' END,
    CASE WHEN cardinality(candidate_ids) = 1 THEN candidate_ids[1] ELSE NULL END,
    to_jsonb(candidate_ids)
  FROM matches;
END;
$$;
-- +goose StatementEnd


SELECT refresh_disease_taxonomy_migration_report();
DROP TABLE medical_guidelines;

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
  RAISE EXCEPTION 'Medical guideline retirement is irreversible. Restore the pre-migration database backup to recover removed content.';
END $$;
-- +goose StatementEnd

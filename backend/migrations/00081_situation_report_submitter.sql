-- +goose Up
-- Records who submitted a report for review, so the approver can be required
-- to be someone else (as well as someone other than the author).
ALTER TABLE situation_reports
  ADD COLUMN submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN submitted_at TIMESTAMPTZ;

-- Reports already submitted take their latest submission from the audit log.
UPDATE situation_reports r
SET submitted_by = s.actor_id, submitted_at = s.created_at
FROM (
  SELECT DISTINCT ON (a.entity_id) a.entity_id, a.actor_id, a.created_at
  FROM audit_logs a
  JOIN users u ON u.id = a.actor_id
  WHERE a.action = 'situation_report.submit'
  ORDER BY a.entity_id, a.created_at DESC
) s
WHERE s.entity_id = r.id::text;

-- +goose Down
ALTER TABLE situation_reports DROP COLUMN IF EXISTS submitted_at, DROP COLUMN IF EXISTS submitted_by;

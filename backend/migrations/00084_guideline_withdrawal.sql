-- +goose Up
ALTER TABLE guideline_documents
  ADD COLUMN withdrawn_version_id uuid,
  ADD COLUMN withdrawn_at timestamptz,
  ADD COLUMN withdrawn_by uuid,
  ADD COLUMN withdrawal_reason text NOT NULL DEFAULT '';

-- +goose Down
-- Do not restore current_version_id: withdrawn documents must remain private.
ALTER TABLE guideline_documents
  DROP COLUMN withdrawal_reason,
  DROP COLUMN withdrawn_by,
  DROP COLUMN withdrawn_at,
  DROP COLUMN withdrawn_version_id;

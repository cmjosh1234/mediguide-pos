-- +goose Up
-- A published correction replaces its original; before this, publishing a
-- correction of an update left both versions public.
UPDATE outbreak_updates AS original
SET status = 'withdrawn',
    withdrawn_at = now(),
    withdrawal_reason = 'superseded by approved correction',
    lock_version = original.lock_version + 1,
    updated_at = now()
FROM outbreak_updates AS correction
WHERE correction.supersedes_id = original.id
  AND correction.status = 'published'
  AND correction.deleted_at IS NULL
  AND original.status = 'published'
  AND original.deleted_at IS NULL;

-- +goose Down
-- Retired originals are not republished; the correction remains the live version.
SELECT 1;

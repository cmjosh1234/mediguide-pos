-- +goose Up
-- A hub's "Situation reports" section used to be filled once, when the hub
-- was set up, so reports published later never appeared on it. Published
-- reports are now added as they are published; this adds the ones published
-- before that to the hubs assigned to their outbreak. Items take the report's
-- own title and summary, and corrections are left to replace their original's
-- item when published.
INSERT INTO content_pillar_items (pillar_id, content_type, content_id, sort_order, status)
SELECT p.id, 'situation_report', r.id, 0, 'active'
FROM situation_reports r
JOIN content_hub_outbreaks cho ON cho.outbreak_id = r.outbreak_id
JOIN content_hubs h ON h.id = cho.content_hub_id AND h.deleted_at IS NULL
JOIN content_pillars p ON p.hub_id = h.id AND p.slug = 'situation-reports' AND p.deleted_at IS NULL
WHERE r.deleted_at IS NULL
  AND r.status = 'published'
  AND r.published_at IS NOT NULL
  AND r.withdrawn_at IS NULL
  AND r.supersedes_id IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM content_pillar_items i
    WHERE i.pillar_id = p.id AND i.content_type = 'situation_report' AND i.content_id = r.id AND i.deleted_at IS NULL
  );

-- +goose Down
-- Backfilled items can't be told apart from ones editors added, so they stay.
SELECT 1;

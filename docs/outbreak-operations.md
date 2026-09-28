# Outbreak operations runbook

For the end-to-end dashboard sequence that creates an outbreak hub, links its
typed resources, assigns pillar items and verifies mobile visibility, use
the [content hub and document publishing workflow](content-hub-and-document-publishing-workflow.md).

## Discovery and freshness

Public search returns only published, non-withdrawn guidelines, outbreaks, and situation reports. Results use the explicit `result_type` values `guideline`, `outbreak`, and `situation_report`. Search covers approved editorial fields: title, summary, disease, geography, source organization, source reference, and situation-report highlights.

The mobile app also searches its public outbreak/report cache while offline. Cached results are clearly marked. Stale results never receive the active/recent ranking bonus and are labelled “verify when online.” The app emits `outbreak_cache_access` (`hit`/`miss`) and `outbreak_offline_content_used` through Firebase Analytics with only `content_type`, cache result, and a numeric `stale` flag; it sends no user ID, token, query, or clinical content. Firebase queues analytics locally while the device is offline and uploads them after connectivity returns.

## Metrics and privacy

Prometheus-compatible metrics are exposed at `GET /api/metrics`. Configure the production reverse proxy so only the monitoring network can reach this endpoint.

- `mediguide_outbreak_public_requests_total`
- `mediguide_outbreak_public_request_errors_total`
- `mediguide_outbreak_public_request_duration_seconds_sum`
- `mediguide_cache_hits_total`, `mediguide_cache_misses_total`, `mediguide_cache_errors_total`
- `mediguide_outbreak_verification_age_hours`
- `mediguide_outbreak_publications_total{state="published|withdrawn"}`
- `mediguide_notification_campaigns_total`
- `mediguide_notification_deliveries_total`
- `mediguide_outbreak_report_asset_failures_total`
- `mediguide_outbreak_malformed_actions_total`
- `mediguide_managed_report_storage_healthy`

HTTP metrics use bounded route and status labels. Resource IDs appear only in structured request logs for incident correlation. User identifiers, tokens, search queries, payloads, and private participant data are never metric labels or outbreak-operation log fields.

Outbreak sources are currently editorially entered and reviewed; the runtime does not automatically fetch external outbreak sources. Consequently there is no source-fetch counter to alert on until a source-ingestion worker exists. Managed report-asset failures are measured and alerted now; any future source worker must emit a bounded `source` classification (never a raw URL) and add a repeated-failure rule before it is enabled in production.

Load [the alert rules](../infra/monitoring/outbreak-alerts.yml) into the platform Prometheus-compatible rule evaluator. Route critical alerts to the public-health editorial on-call and infrastructure on-call.

## Storage and failure response

`GET /api/readyz` verifies PostgreSQL, required Redis, and the managed report bucket. A failed object-store check removes the API from readiness without changing published data. For repeated asset failures:

1. Check `mediguide_managed_report_storage_healthy` and MinIO/S3 connectivity.
2. Correlate `outbreak_request_completed` logs using `outbreak_id` and route.
3. Confirm the report asset row points to an object in the configured managed bucket.
4. Restore the missing object from the controlled backup or upload a corrected report through the typed admin endpoint.
5. Never replace a published report silently; use the correction workflow when its content changes.

## Withdrawal and rollback

Withdrawal is the immediate public rollback mechanism. It preserves audit history and prevents the record from appearing in public APIs or search.

1. Confirm the affected UUID and current lock version in the dashboard.
2. Call `POST /api/v2/outbreaks/:id/withdraw` or `POST /api/v2/situation-reports/:id/withdraw` with a specific reason and the current lock version. The caller needs the corresponding withdraw permission.
3. Verify the public detail returns 404 and search no longer returns the item.
4. If replacement content is required, use `POST .../:id/correct`, complete independent review, then publish the correction. Do not edit or republish the withdrawn immutable record.
5. Verify the audit trail, notification state, public endpoint, cache behaviour, and metrics before resolving the incident.

Database rollback is not a content-withdrawal mechanism. Use database restore only for infrastructure disaster recovery, following backups and migration compatibility checks.

## Quick resources and search operations

Public quick resources are available through `GET /api/public/outbreak-resources`. They cover related published MediGuide guidelines, published situation reports, approved HTTPS official websites, approved official statements and allowlisted internal application routes. SOPs, protocols, checklists and forms are published in the guideline library and linked to an outbreak as typed resources.

Each result identifies its parent outbreak, issuer, resource and target type, publication date, safe target, and whether it supports an in-app reader or download. External websites are visibly labelled in mobile search and require confirmation before the operating-system browser opens. The API rejects non-HTTPS external targets and hosts outside `OUTBREAK_ALLOWED_EXTERNAL_HOSTS`; internal targets must use an approved application route. A related guideline or report target is returned only when its referenced record is currently public.

## Demo and staging seed

Use `SEED_SCOPE=outbreaks` when only the outbreak experience needs test data. It creates the required development actors and publishes the deterministic Bundibugyo virus disease response hub, metrics, updates, quick resources and situation report. The broader `SEED_SCOPE=demo` seed includes the same outbreak fixture with all other demo content.

The public search terms include both the precise disease name and the commonly used outbreak term (`Ebola`), so an outbreak-hub result is returned for either term.

```bash
docker compose \
  --env-file infra/development.env \
  -f infra/docker-compose.yml \
  -f infra/docker-compose.dev.yml \
  run --rm --no-deps -e SEED_SCOPE=outbreaks api /app/seed
```

The focused `outbreaks` scope is always blocked when `APP_ENV=production`. General demo seeding is also blocked there unless an operator explicitly sets `SEED_ALLOW_DEMO=true`. That override is intended only for a controlled demo/staging server whose environment happens to use the production Compose profile. Real production deployments should seed only approved scopes such as `SEED_SCOPE=admin` or `SEED_SCOPE=facilities`, then publish clinician-approved content through the governed dashboard workflow.

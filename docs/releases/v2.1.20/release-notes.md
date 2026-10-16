# MediGuide v2.1.20

Changes since v2.1.19. Mobile version: 2.1.20+73.

## Mobile application

- Read reviewed structured publications even when a raw Markdown fallback is unavailable.
- Keep existing withdrawal availability checks and offline-reading behavior.

## API

- Restore withdrawn publications without requiring discarded ingestion intermediates, while retaining clinical validation and original approvals.
- Create unapproved replacement drafts from published Markdown, reviewed structured content, replacement files or links.
- Preserve source-version lineage and withdrawal audit history; reject stale replacement publication and substantial content loss.

## AI worker

- No service-specific changes since v2.1.19.

## Dashboard

- Revise withdrawn documents or continue an existing draft while preserving the withdrawn source for inspection.
- Suggest editable next version numbers, avoid existing labels, and show omitted content before review.
- Fix document-history navigation and keep withdrawn versions read-only with original-file previews.

## Public guidelines portal

- No service-specific changes since v2.1.19.

## Deployment and operations

- Synchronize platform version 2.1.20 and Android build 73.
- Apply migration 85 through the standard production deployment workflow before serving the updated API and dashboard.
- Do not run demo seeds in production.
- Server deployment requires successful public storage preflight.
- Google Play public rollout remains gated by recorded submission evidence; Firebase Android and the Play candidate use the production configuration.

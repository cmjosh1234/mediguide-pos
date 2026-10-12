# MediGuide v2.1.17

Changes since v2.1.16. Mobile version: 2.1.17+70.

## Mobile application

- Add reliable notification catch-up, automatic retry, offline inbox access and consistent read state.
- Add notification interests, followed guidelines, daily digests and device permission/registration status.
- Show guideline version changes and saved-copy status; add update checks and actionable foreground alerts.
- Refresh Linux guideline-reader screenshot baselines for the new follow controls and reader state.

## API

- Apply migration 82 for notification interests and durable daily digest batches.
- Deliver outbreaks through consent-aware per-user inbox campaigns; preserve urgent alert handling.
- Add optional Firebase BigQuery delivery ingestion and private notification-worker metrics.

## AI worker

- No service-specific changes since v2.1.16.

## Dashboard

- Allow editors to include reviewed change summaries in guideline notification drafts.
- Clarify provider delivery reporting and digest delivery counts.

## Public guidelines portal

- No service-specific changes since v2.1.16.

## Deployment and operations

- Publish unified version 2.1.17 and Android build 70 with ARM and x86_64 native library checks.
- Wire optional Firebase delivery importer settings into the notification worker environment.
- Provide notification queue/importer monitoring rules and deployment/acceptance instructions.

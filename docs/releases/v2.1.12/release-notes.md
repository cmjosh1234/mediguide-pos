# MediGuide v2.1.12

Changes since v2.1.11. Mobile version: 2.1.12+65.

## Mobile application

- Browse content hubs and conditions with clearer layouts, compact resource cards, and useful empty states.
- Read situation reports with clearer indicators and expandable summaries.
- Use a single calculator header and improved clinical tool input fields.
- Restore the red debug badge with vertical-only movement in development and staging builds.

## API

- Align clinical tool JSON definitions and input fields with the source forms.

## AI worker

- No worker-specific behavior changes since v2.1.11; version synchronized to 2.1.12.

## Dashboard

- Add missing empty states and recovery actions.
- Resolve the missing countries dependency in the development container.

## Public guidelines portal

- No portal-specific behavior changes since v2.1.11; version synchronized to 2.1.12.

## Deployment and operations

- Generate release notes for all services and versioned Google Play changelogs.
- Production Android uses the hosted production API. Google Play upload creates a production-track draft; public rollout remains separate.
- Signed iOS store delivery requires Apple credentials that are not currently configured.

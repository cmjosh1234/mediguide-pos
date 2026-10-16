# MediGuide v2.1.19

Changes since v2.1.18. Mobile version: 2.1.19+72.

## Mobile application

- Remove withdrawn guideline caches and saved downloads when availability is checked at startup or on reconnection.
- Preserve verified offline copies during connection and server failures.

## API

- Add audited document withdrawal with a required reason while retaining files, approved versions and review history.
- Republish through existing validation checks while preserving the original approval and recording republication.
- Require public-response revalidation so withdrawn publications are not reused from browser caches.
- Expand development fixtures to cover workflow history tables and preserve withdrawals when reseeding.

## AI worker

- No service-specific changes since v2.1.18.

## Dashboard

- Add Withdraw and Review and republish actions, withdrawal status, actor, time and reason.
- Keep document deletion separate from withdrawal and reject stale withdrawal requests.

## Public guidelines portal

- Refresh publication cards and document-kind counts when returning to the tab and every 30 seconds while visible.
- Bypass stale browser API responses and honor configured API endpoints in local development.

## Deployment and operations

- Synchronize platform version 2.1.19 and Android build 72.
- Apply migration 84 through the standard deployment workflow before serving the updated API and dashboard.
- Release the updated mobile app for startup and reconnect removal of withdrawn downloads. Devices must reconnect to learn of withdrawals; exported files cannot be recalled.
- Expanded seed records are synthetic development fixtures. Do not run the demo seed in production; account and notification examples are terminal histories and do not enqueue external messages.
- Keep using the standard tagged release workflows for production deployment, Firebase delivery and the Google Play draft.

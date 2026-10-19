# MediGuide v2.1.21

Changes since v2.1.20. Mobile version: 2.1.21+74.

## Mobile application

- No service-specific changes since v2.1.20.

## API

- Show actionable search-indexing timeout guidance while preserving the saved Markdown revision.

## AI worker

- Recover slow Ollama embedding batches by splitting timed-out requests while preserving full inputs and order.
- Honor OLLAMA_EMBEDDING_TIMEOUT_SECONDS and retain successful embedding checkpoints across retries.

## Dashboard

- No service-specific changes since v2.1.20.

## Public guidelines portal

- Remove the open staff dashboard button from the public guidelines portal.

## Deployment and operations

- Use one embedding request and one document worker at a time in production, with a single-text batch and a 600-second embedding deadline.
- Expose EMBEDDING_REQUEST_BATCH_SIZE to both AI services through Compose and default older deployment bundles to single-text batches without replacing existing tuning or credential values.
- Deploy all service images under the immutable 2.1.21 tag; synchronize the protected PRODUCTION_ENV_FILE secret with the updated private production environment before deployment.

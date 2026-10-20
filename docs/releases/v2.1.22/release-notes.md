# MediGuide v2.1.22

Changes since v2.1.21. Mobile version: 2.1.22+75.

## Mobile application

- No service-specific changes since v2.1.21.

## API

- No service-specific changes since v2.1.21.

## AI worker

- No service-specific changes since v2.1.21.

## Dashboard

- No service-specific changes since v2.1.21.

## Public guidelines portal

- No service-specific changes since v2.1.21.

## Deployment and operations

- Increase the 8-vCPU/32-GiB production processing profile to two concurrent documents with three PDF table-extraction workers per document; keep one queue-worker replica initially.
- Use two OCR workers and one embedding worker per document, four-text embedding batches, a 600-second embedding deadline and reusable artifact checkpoints.
- Synchronize the protected production environment with the throughput profile before deployment; monitor memory, API responsiveness, job duration and embedding retries, and reduce document concurrency if contention increases.
- Publish all service images under the immutable 2.1.22 tag. Include the five-slide MediGuide platform overview in the repository.

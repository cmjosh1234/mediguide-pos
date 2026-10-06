# MediGuide v2.1.11

Changes since v2.1.10. Mobile version: 2.1.11+64.

## Mobile application

- Use clinical calculators and reference directories without signing in.
- Read cleaner guideline text with improved paragraph alignment and fewer visible HTML fragments.
- Improved guideline-reader layout with updated regression coverage.

## API

- Added public routes for clinical calculators and reference directories.
- Updated calculator handling and covered unauthenticated tools access with regression tests.

## AI worker

- No worker-specific behavior changes since v2.1.10; version synchronized to 2.1.11.

## Dashboard

- Updated frontend dependencies and aligned the dependency lockfile to restore CI checks.
- Updated the dashboard container runtime configuration.

## Public guidelines portal

- Improved Markdown paragraph presentation and text alignment in the public guideline reader.

## Deployment and operations

- Published immutable version 2.1.11 images for the API, AI worker, dashboard and guidelines portal.
- Published signed Android APKs and AAB for mobile version 2.1.11+64.
- Production deployment and public health checks passed. Google Play accepted build 64 as a production-track draft; public rollout remains separate.

# MediGuide v2.1.13

Changes since v2.1.12. Mobile version: 2.1.13+66.

## Mobile application

- Improve account verification and password-reset handling.

## API

- Track account creation, verification and password reset with a durable email queue.
- Add Resend email delivery with bounded requests, private errors and idempotent retries.

## AI worker

- Document and align standalone runtime environment settings.

## Dashboard

- Complete verification and password-reset flows and account lifecycle analytics.

## Public guidelines portal

- Resolve dashboard login URLs without duplicating the /admin/login path.

## Deployment and operations

- Align environment schemas, preserve populated credentials and support Resend production configuration.
- Validate public storage routing before replacing the production stack.

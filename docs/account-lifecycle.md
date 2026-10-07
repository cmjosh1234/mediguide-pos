# Account lifecycle

Account creation, password reset, email verification, and administrative approval are separate operations. Unverified users retain the public content access allowed by the existing access policy.

## Behaviour

- Self-registration validates email, name, and an password of at least eight characters (maximum 72 UTF-8 bytes, matching bcrypt) containing a letter and number. A transaction creates the account, audit events, a hashed 24-hour verification token, and an encrypted email queue record. A failed email send does not undo account creation. Mobile distinguishes successful account creation followed by failed sign-in and directs the user to sign in.
- Admin-created accounts use the same verification queue and creation audit. Admin approval continues to use `verified`; email ownership uses `email_verified`. Admin user lists, profile badges, exports, and approval/resend actions show these statuses separately. Email confirmation never grants administrative approval.
- Reset requests return the same account-neutral response for known and unknown addresses. Eligible accounts receive a one-hour, single-use reset token. Confirmation validates the password, consumes the token, changes the password, revokes every session, and records completion in one transaction. Mobile clears its stored session; the dashboard clears its auth store.
- Verification links require an explicit confirmation action, preventing page loading and email scanners from consuming tokens. Expired or used links offer resend, with a 60-second client cooldown and backend rate limits.
- Changing an account email clears email ownership, invalidates outstanding account-action links, and queues verification for the new address. Admin password updates invalidate pending reset links and revoke sessions. Account updates and token operations lock the account in PostgreSQL to serialize concurrent changes.
- Browser account pages provide an “Open in MediGuide app” button using `mediguide://account/reset-password?token=...` or `mediguide://account/verify-email?token=...`. Both Android and iOS register the scheme. These mobile routes bypass onboarding and authentication redirects so the token survives a cold launch.

## Delivery

The API drains the durable account email queue on startup and every 30 seconds. Compare-and-set leases allow multiple API replicas to share work. A crashed worker's lease expires after two minutes. SMTP and Resend calls have bounded operation timeouts and respect cancellation. Remote SMTP servers must support STARTTLS; plaintext localhost SMTP is available for tests.

Failed delivery attempts retry after 2, 4, 8, and 16 minutes, then enter `failed` on the fifth failed attempt. Expired, consumed, superseded, or deleted-account tokens enter `cancelled`. Terminal records clear their encrypted token payload. Raw tokens are never written to audit metadata. The development mail driver is rejected in production.

The queue encrypts raw email tokens using a key derived from `JWT_SECRET`; keep that secret stable across replicas. Changing it can invalidate pending queue payloads; users can request a new link. Delivery is at least once: a worker crash after provider acceptance but before the database commit can cause a retry. The Resend driver sends a stable `account-email/<delivery-id>` idempotency key across retries and lease recovery. Resend retains keys for 24 hours; a recovery beyond that window can still duplicate a message. `sent` means provider acceptance, not confirmed inbox delivery. Bounce/spam status requires provider integration and cannot be inferred from acceptance. [Resend idempotency documentation](https://resend.com/docs/dashboard/emails/idempotency-keys)

### Resend production configuration

The `resend` mail driver sends plain-text account emails through
`https://api.resend.com/emails`. It validates the sender/key configuration,
requires a successful response with a message ID, rejects redirects, and
never logs provider response bodies or the API key. Provider failures enter
the existing durable retry flow. No additional SDK or database migration is
needed for this adapter. [Resend send-email API](https://resend.com/docs/api-reference/emails/send-email)

1. Verify `mediguide.health.go.ug` (or the chosen sending subdomain) in Resend
   and add the DNS records it supplies.
2. Create an API key with sending permission for that domain. Store it privately
   in the production env/deployment secret; do not put it in a public GitHub variable.
3. Configure:

   ```dotenv
   MAIL_DRIVER=resend
   MAIL_FROM="MediGuide <no-reply@mediguide.health.go.ug>"
   RESEND_API_KEY=<set privately>
   ACCOUNT_ACTION_URL=https://mediguide.health.go.ug/admin
   ```

4. Deploy the new backend image and recreate the API using the production and
   release env files. SMTP fields are unused by this driver. Existing populated
   env files retain their previous driver until credentials are configured.
5. Register a test account, verify the email, request a password reset, and
   confirm the links open the correct `/admin` pages. Check provider acceptance
   in Account Analytics and inbox delivery in Resend and the recipient mailbox.

SMTP remains supported with `MAIL_DRIVER=smtp`. To use Resend's SMTP relay
instead of its API, set `SMTP_HOST=smtp.resend.com`, `SMTP_PORT=587`,
`SMTP_USERNAME=resend`, and set `SMTP_PASSWORD` privately to the API key.
The HTTPS adapter supplies the provider idempotency header automatically.
[Resend SMTP documentation](https://resend.com/docs/send-with-smtp)

## Tracking and reporting

The dashboard's **Account Analytics** page (`/admin/analytics/accounts` in production) loads `/api/v2/analytics/accounts?days=30`. The endpoint requires `analytics.read` or `admin.all`, uses authenticated rate limits, and returns private, non-cacheable aggregates without user emails, IDs, or tokens. The lookback accepts 1–90 days. Queue status totals cover the retained queue, while event counts use the selected lookback.

Server audit events track `user.account_created`, `user.password_reset_requested`, `user.password_reset_completed`, `user.email_verification_requested`, `user.email_verified`, and `user.verified`. Queue events track send attempts, provider acceptance, terminal failure, and cancellation. Failed send attempts and terminal failed emails are separate counters. Accepted public requests for unknown addresses do not create account-specific audit events.

Mobile emits signup attempt/success/failure and reset/verification attempt/success/failure events through the existing Firebase operational telemetry. Those events contain only event names and surface/method labels. Telemetry failures never block account actions. Account-action routes are excluded from generic content usage tracking, and tokens are not included in events.

## Deployment and validation

1. Apply migration `00078_account_lifecycle.sql` before deploying the new API. It adds `users.email_verified` and the account email queue. Historical `user.email_verified` audit events backfill known email ownership. Existing admin approval is preserved because expired audit history cannot prove that approval was absent; review ambiguous legacy approvals separately.
2. Set `ACCOUNT_ACTION_URL` to the full externally reachable dashboard auth base, e.g. `https://mediguide.health.go.ug/admin`. Legacy `PUBLIC_APP_URL` root and `/admin/login` configurations also resolve correctly. Configure the Resend driver above or use `MAIL_DRIVER=smtp` with valid SMTP host, port, sender, and credentials. Do not enable development mail in production.
3. Deploy the API and dashboard, then release mobile builds containing the native URL scheme registration. Restarting Flutter with hot reload alone does not apply native manifest changes.
4. Using a dedicated test account, register and check the queue, open verification in browser and mobile, request a reset, confirm it, and verify old sessions and reused links fail. Confirm the matching lifecycle counters increase. Check spam/inbox separately from provider acceptance.

Automated coverage includes registration validation and partial success, encrypted queue persistence and retry limits, lease recovery, expired/superseded/replayed links, email changes, session revocation, aggregate privacy and input validation, local SMTP acceptance/rejection/cancellation, mobile account screens at small width and 200% text, dashboard confirmation/resend and analytics recovery, and profile screenshots for macOS and Linux.

These source changes do not themselves deploy a production migration, web release, or mobile store update. Live inbox delivery and native deep-link behaviour on installed release builds must be checked after deployment.

# Notification experience and production setup

## Behavior

- The backend inbox is the durable record. Push is a prompt to revisit that record.
- Published outbreak jobs now fan out once into per-user campaigns. They no longer
  send to the public Firebase topic. Existing topic subscriptions are harmless;
  older mobile releases still receive the new device-targeted notifications.
- Outbreak alerts honor the outbreak category. Urgent outbreak alerts additionally
  honor the emergency category. Only urgent emergency alerts bypass quiet hours;
  they never bypass category, account or device opt-outs.
- Routine interests filter program areas, selected regions and followed guidelines.
  Empty program/region lists include all. National outbreak notices match every
  region; urgent alerts bypass routine interest filters, subject to category consent.
  Followed-only filtering is opt-in. Following one guideline does not implicitly
  turn off updates about every other guideline. Explicitly followed guidelines
  match regardless of the selected routine program areas.
- Push text is a bounded preview; full editorial text and change summaries remain
  in the inbox. Display metadata does not become a navigation query string.
- Routine foreground messages use a dismissible in-app banner. High-priority and
  urgent messages retain the system notification channel. Message IDs prevent
  repeated foreground displays. Routine banners are suppressed when viewing their
  exact target route.
- Inbox catch-up runs on startup, resume, reconnect and incoming foreground push.
  Automatic successful checks have a one-minute throttle; failures have a short
  backoff with automatic foreground retries, capped at five minutes. Background
  retries pause until resume. Explicit checks and reconnect bypass the throttle.
  Requests are serialized.
- A push open updates its linked inbox read state. Opening content is independent
  of read-state or engagement-upload failures. The inbox preserves unsynced local
  read/unread changes when an online response arrives.
- The inbox identifies saved/offline content and the last successful sync. Cached
  empty inboxes and empty final pages are valid offline results.
- iOS badges are synchronized to the unread count while the app is active and reset
  on logout. Push payloads no longer assign an arbitrary badge of one. The badge
  may remain stale while the app is closed; it is not a background unread counter.
- Both Android and APNs receive expiry settings. Unspecified transport lifetimes
  default to 24 hours. A queued job's remaining expiry caps its transport lifetime.
- Remote Config fetches do not block notification initialization or offline access.
  Flag providers react to initial and real-time activation even when the outbreak
  banner value itself remains unchanged.

## Daily digest

Users enable this in Notification settings and choose a time in their quiet-hours
timezone. Routine low/normal-priority updates remain in the inbox immediately and
are combined into one summary per device and delivery window. High/urgent alerts
are sent through the normal immediate delivery path.

Digests require both an enabled inbox and an in-app campaign channel; push-only
campaigns retain individual delivery so the summary never points at an empty inbox.
The worker rechecks expiry, consent and interests before sending. Turning digest
mode off releases pending members to the normal delivery worker. Campaigns paused
for editorial review are not sent by the digest worker. A late schedule or explicit
requeue can reopen a completed window; retries may redeliver a summary, as with
any FCM send accepted before a worker/database failure. The collapse key and mobile
message-ID deduplication reduce repeated displays; FCM is not exactly-once delivery.

A digest contains multiple per-campaign lifecycle records but one provider send.
Delivery export can confirm delivery for all its members. Its inbox action does
not count every constituent guideline as clicked; individual inbox opens/clicks
remain the meaningful content-engagement measurements.

## Guideline updates

Editors can add a bounded **What changed?** summary when creating a guideline
notification draft. The existing review/approval workflow remains in place. The
notification carries the publication version and reviewed summary into the inbox.
Mobile readers can follow guidelines from the reader toolbar or Notification
settings. The reader shows the saved/current version and offers an offline-copy
update. Notification cards identify whether a copy is saved and requires review.

## Deploy order

1. Apply migration `00082_notification_experience.sql` with the normal backend
   migration job. It adds interests, digest settings/batches and delivery-export
   event support.
2. Deploy the API and notification worker together. Update the dashboard to expose
   the change-summary field.
3. Release the mobile client. Older clients still support the existing push
   channels/actions; personalization and the improved inbox require the new client.
4. Configure delivery export below when Google Cloud setup is ready. Push and the
   inbox operate independently of export ingestion.

No production console settings or production server credentials are changed by
these source changes.

## Firebase delivery reporting

Accepted means Firebase accepted the send. Delivered requires an imported
`MESSAGE_DELIVERED` event or an authenticated client open/click. A missing export
record is unknown, not proof that delivery failed. Export and metrics can be
delayed, and diagnostics-disabled devices are not fully represented.

1. In the **mediguide-production** Firebase project, enable Cloud Messaging export
   under the BigQuery integration. Confirm the actual table and its region.
   The conventional table is `mediguide-production.firebase_messaging.data`.
2. Create a separate importer service account with **BigQuery Job User** on the
   query/billing project and **BigQuery Data Viewer** on the export dataset. Do not
   grant write access. Enable the BigQuery API on the query project.
3. Set the notification worker's production environment:

   ```dotenv
   FCM_DELIVERY_BIGQUERY_TABLE=mediguide-production.firebase_messaging.data
   FCM_DELIVERY_APP_NAME=com.mediguide.ug
   FCM_DELIVERY_SERVICE_ACCOUNT_BASE64=<base64-encoded importer service-account JSON>
   ```

   The table must belong to the sending Firebase project. `FCM_DELIVERY_APP_NAME`
   accepts a comma-separated list of exact application identifiers. Include the
   deployed iOS bundle identifier when shipping iOS; this repository currently
   builds the simulator as `com.omarsoft.mediguide`. Do not assume it shares the
   Android package identifier or is registered in the production Firebase project.
   An empty table disables importing. A separate account is preferred; application
   default credentials are also supported when the importer credential is empty.
   The backend's FCM sending account remains `FIREBASE_SERVICE_ACCOUNT_BASE64`.
   Firebase App Distribution upload credentials do not establish backend push access.
4. Restart the worker with the updated environment. Import starts immediately and
   then repeats every six hours with a seven-day overlap. Replays are idempotent.
   Each query is capped at 1 GiB of scanned data; large datasets may need a smaller
   partition window or an explicitly reviewed budget change.
5. Users explicitly enable **Share delivery diagnostics** in Notification settings.
   It is off by default. Android enables the SDK export; iOS exports when the native
   remote-notification callback runs. iOS background execution and OS settings affect
   coverage; validate on physical devices before interpreting Apple delivery rates.
6. Check worker import success and the dashboard's delivery audit. The importer only
   matches the configured application, Firebase sender project and device platform.
   Ambiguous provider IDs spanning multiple installations remain unknown; they
   are not credited to the wrong device. Provider export ingestion is internal; clients cannot assert provider delivery.

Official references:
- https://firebase.google.com/docs/cloud-messaging/understand-delivery
- https://firebase.google.com/docs/cloud-messaging/flutter/receive-messages
- https://firebase.google.com/docs/cloud-messaging/customize-messages/setting-message-lifespan

## Monitoring

The internal notification worker exposes `/metrics` on its existing port (8082 by
default). Keep this on the private service network. Configure your Prometheus
scraper with job name `mediguide-notifications` for `notification-worker:8082/metrics` and load
`infra/monitoring/notification-alerts.yml` into the existing monitoring service.

The rules cover worker availability, overdue delivery/publication jobs, technical delivery failures, missing Firebase
configuration and stale delivery-export imports. Import health is independent of
worker liveness: reporting outages must not stop inbox or push processing. Confirm
FCM sending is configured through the admin Firebase status page as well as worker
metrics; a healthy database-only readiness response does not prove FCM access.

## Physical-device acceptance checks

Use a consenting tester and the admin notification workflow, not a general user
audience. Complete on a supported Android device and an iPhone before rollout:

| Scenario | Expected result |
| --- | --- |
| Routine push while app is open | One dismissible in-app banner; no OS interruption |
| Urgent opted-in alert while app is open | Emergency channel notification and durable inbox record |
| Background and cold-launch push tap | Correct target opens; linked inbox record becomes read |
| Lost push / normal resume | Inbox catches up without requiring another push |
| Offline inbox/read action, then reconnect | Saved data remains usable; read change survives and synchronizes |
| Permission denied or changed in Settings | Honest status and Allow/Settings recovery; no startup failure |
| Account or device push opt-out | No targeted push; inbox follows its separate opt-in |
| Quiet hours | Routine push deferred; only opted-in urgent emergency alerts bypass |
| Two routine updates with digest enabled | Inbox gets both; one daily summary arrives at the selected time |
| Expired digest member / withdrawn resource | Old member not sent; unavailable content gives the normal reader error |
| Logout and account switch | Old local banners/badge cleared; new installation registration belongs to the new account |
| Updated followed guideline | Version and editorial changes visible; old saved copy can be updated |
| Diagnostics off/on | Export disabled/enabled for this installation; UI accurately reflects the saved choice |
| Retry/import replay | No duplicate inbox records or inflated delivery counts |

Automated coverage includes offline merges/pagination, refresh throttling/failures,
preference validation, digest grouping/expiry/opt-outs, outbreak fanout, platform
expiry payloads, delivery-export replay/ownership and the existing deep-link tests.
Physical FCM/APNs receipt, OS permission behavior and Google Cloud integration cannot
be established by unit tests or source inspection.

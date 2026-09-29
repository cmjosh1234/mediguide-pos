# Account deletion operations and release evidence

Status: request intake implemented; fulfilment is not automated or certified.
Do not describe this change as completed erasure or mark the release ready.

## Entry points
- Mobile: Profile > Delete Account > current password and explicit confirmation.
- Public portal: /delete-account; existing email/password required, no app install.
- POST /api/v2/me/deletion-request: authenticated; current_password and confirm=true.
- POST /api/v2/auth/deletion-request: email, current_password and confirm=true.
- HTTP 202: data.request_id and data.status=requested. No deletion occurs on GET.
- The existing dashboard support queue receives category account_deletion, priority high.
- Active requests deduplicate per user under a database row lock.
- Generic support creation cannot claim this reserved category. Requests cannot be
  rewritten by account holders or removed through the generic ticket deletion endpoint.
- Wrong authenticated password is a 400 response to avoid triggering session expiry.
- No password is written into a ticket or audit record.

## Before enabling production
Assign an accountable support owner and define a processing target and escalation route.
Confirm recovery/support for disabled accounts and users unable to reset their password.
Verify that credentials/request bodies are never captured in access logs, analytics or traces.
Test correct/wrong passwords, retries, rate limits, deleted/inactive users, ownership and staff access.
Publish a verified privacy contact, processing period, retention exceptions and backup expiry.
The web privacy page intentionally remains a draft until these facts are supplied.

## Fulfilment scope
The request alone is insufficient for Google Play compliance. The operator must actually:
1. Verify the request/audit record and investigate any genuinely required retention.
2. Remove the account's authentication material and revoke all access/refresh sessions.
3. Remove personal profile/contact details, saved notes, reading history, bookmarks,
   subscriptions, usage data, device registrations/tokens, AI chats and retrieval logs.
4. Assess support messages, peer conversations and attachments without destroying
   another user's records or shared clinical publications.
5. Remove/request removal from relevant processors, AI services and Firebase as applicable.
6. Address backups under the approved schedule and prevent restored data reactivating accounts.
7. Confirm completion and clearly explain any retained data, reason and duration.
8. Record evidence and only then close the request. A ticket marked resolved does
   not itself execute any deletion.

Existing foreign keys include RESTRICT relationships for notifications, calculator authors,
support and conversations. Do not run a blanket DELETE against users in production.
A schema-aware erasure implementation and production-like database rehearsal remain required.

## Release gating
play-store/release-readiness.json tracks outstanding evidence. All required checks need
status=passed plus concrete evidence before --submission succeeds. Do not mark checks passed
on the basis of generated source or a green packaging check. Keep secrets and reviewer
credentials out of this file.

## References
https://support.google.com/googleplay/android-developer/answer/13327111
https://support.google.com/googleplay/android-developer/answer/10144311

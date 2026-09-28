# Data safety and health declaration worksheet
DRAFT: complete with the data controller and backend/AI owners before submitting.
Do not declare "no data collected": source enables Firebase Analytics and Crashlytics,
including a user identifier in crash reporting, and supports authenticated services.

| Data / source | Evidence and questions to resolve |
| --- | --- |
| Name, email, user ID and professional profile | Account/profile features. Confirm mandatory fields, purposes, deletion and retention. |
| Facility/organization/professional context | Listed in in-app terms. Confirm actual payloads and whether optional. |
| App interactions, searches | Firebase Analytics collection is enabled. Inventory events and any search telemetry. |
| Crash logs, diagnostics, device identifiers | Crashlytics and messaging. Review SDK disclosures and actual configuration. |
| Push token | Device registration/notifications. Confirm linkage to account and removal on logout/deletion. |
| Bookmarks, notes and reading progress | Confirm what syncs to the backend versus stays only on device. |
| AI queries, conversation history and user content | Confirm provider, countries, retention, logging and whether users can enter health/patient information. Do not classify prompts as anonymous without proof. |
| Downloads and preferences | Determine local-only versus synchronized data and backup behavior. |
| Biometric authentication | Verify local OS authentication only; do not imply the app collects raw biometrics without evidence. |

For each applicable Play data type specify collected/shared, purpose, required/optional,
ephemeral processing, encryption in transit, and deletion support. Evaluate service-provider
exceptions using Google's definitions; a third-party SDK does not automatically imply sharing.
Verify server TLS and SDK behavior before affirming encryption in transit.

Health declaration: describe guideline reference, clinical tools/calculators and AI-assisted
retrieval accurately. Select all applicable Console categories, including clinical decision
support if the released functionality fits; the owner must validate regulatory classification.
Do not represent the product as an approved medical device without supporting evidence.

Privacy policy publication checklist:
operator identity; support/privacy contact; data types and purposes; processors and AI providers;
international processing where applicable; retention and backups; deletion request method and
exceptions; security practices; children's policy; effective date; change notifications.
Publish an accessible HTML page and link it in-app and in Play Console.
Do not invent retention periods, processor contracts, legal bases or support addresses.

AI safety: verify citations and limitations, in-app reporting/flagging of inappropriate AI
content, error/fallback behavior and handling of patient-identifiable input. Report missing
safeguards before launch. Have clinical owners validate calculator content and intended use.

Account deletion acceptance: a user can initiate a real request in the app and from the public
web URL; identity verification is proportionate; the backend and identity provider complete
deletion; tokens are invalidated; associated notes/bookmarks/device registrations are addressed;
any legally retained data and backup periods are explained; the user receives confirmation.

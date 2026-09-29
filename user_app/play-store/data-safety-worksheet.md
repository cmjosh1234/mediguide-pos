# Data safety and health declaration worksheet

This is an implementation inventory, not approval to submit the declaration.
The data controller and backend/AI owners must confirm retention, deletion,
processor terms and the final Play answers. Do not declare "no data collected":
the app supports accounts and synchronized services and enables Firebase
Analytics, Crashlytics and Messaging in configured production builds.

## Verified implementation inventory

| Data / source | Current implementation | Likely purpose / handling | Owner decision still required |
| --- | --- | --- | --- |
| Name, email, internal user ID and professional profile | Registration/profile APIs send these fields to the MediGuide backend. | Account management, authentication and personalization. | Required/optional fields, retention, deletion exceptions and whether any profile fields are public. |
| Facility, organization and professional context | Supported by account/profile records and authenticated workflows. | Role and facility context. | Confirm exact production fields and whether each is optional. |
| App interactions and search activity | Firebase Analytics is enabled. The current explicit operational event records global-search completion metadata; Firebase also creates an app-instance identifier under its SDK behavior. Production disables Android Advertising ID collection and ad-personalization signals. | Analytics and product reliability; not advertising. | Approve the complete Analytics event inventory and retention configuration in Firebase. |
| Crash logs and diagnostics | Crashlytics receives uncaught and explicitly reported non-fatal failures plus build/platform context. The app always clears the Crashlytics user identifier, so a MediGuide account ID is not attached. | App stability and diagnostics. | Confirm Firebase retention and access controls. |
| Push token and installation identifier | After notification consent and sign-in, the app sends an FCM token, random app installation ID, platform, app version and locale to the MediGuide backend. The device record and FCM token are removed during app logout/unregistration. | Deliver opted-in notifications. | Confirm server retention, deletion fulfilment and Firebase Messaging disclosure. |
| Bookmarks, private notes and reading progress | Reading-progress records, bookmark state and notes synchronize to the authenticated backend and are cached locally for offline use. | User-requested library and cross-device continuity. | Confirm retention and account-deletion treatment. |
| AI questions, sessions, citations and retrieval logs | Questions are sent to MediGuide RAG endpoints. Backend models persist chat session/message content and retrieval logs; the inspected production configuration uses self-hosted Ollama models. The UI warns users not to enter patient-identifiable data. | Provide citation-first guideline assistance, safety review and operations. | Confirm production logging/retention, administrator access, backup expiry, and whether any alternate provider can be enabled in production. |
| Support conversations and account-deletion requests | Support tickets/messages are stored by the MediGuide backend. A deletion request creates a verified high-priority support ticket but does not yet erase the account. | User support and deletion-request intake. | Approve fulfilment procedure, service level, retention and confirmation process. |
| Downloads, local cache and preferences | Guideline/offline content, preferences and pending synchronization records are stored on device. Production removes broad storage/media permissions and uses app-private storage. | Offline reading and settings. | Confirm OS/cloud-backup policy and local purge behavior at logout/deletion. |
| Biometric authentication | Uses the operating-system authentication prompt; source does not collect raw biometric templates. | Optional local access protection. | Confirm the Play answer remains consistent with the released configuration. |

For each applicable Play data type specify collected/shared, purpose, required/optional,
ephemeral processing, encryption in transit, and deletion support. Evaluate service-provider
exceptions using Google's definitions; a third-party SDK does not automatically imply sharing.
Verify server TLS and SDK behavior before affirming encryption in transit.

Health declaration: describe guideline reference, clinical tools/calculators and AI-assisted
retrieval accurately. Select all applicable Console categories, including clinical decision
support if the released functionality fits; the owner must validate regulatory classification.
Do not represent the product as an approved medical device without supporting evidence.

Privacy policy publication checklist:
operator identity; approved support/privacy contact; data types and purposes; processors and AI providers;
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

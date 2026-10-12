# Mobile application updates

Android app binaries update through the installation channel. Remote Config
changes feature settings and does not install new application code. This
implementation keeps clinical content available during discovery and download;
installation and restart require the user's consent.

## Google Play installations

- Release-mode production builds check after the first rendered frame and on
  resume, without awaiting the network during app startup.
- A successful availability check is cached for 24 hours. Failed automatic
  checks stay silent and retry after 15 minutes. Profile → Check for Update
  bypasses this cooldown.
- Normal updates offer a dismissible Update notice. The user can continue
  reading while a flexible update downloads. Cancellation never starts a
  fullscreen update as a fallback.
- Once downloaded, offer Restart. Persist the pending download so another
  process launch or resume can recover its status even during the daily
  cooldown. Profile also offers Restart to update.
- Play release priorities 4–5 use the immediate update flow only after the user
  taps Update, and only when Play allows it. Lower priorities prefer flexible
  updates; if that flow is unavailable, offer the Play listing.
- A package installed outside Google Play is not reported as up to date merely
  because Play cannot check it. Generic sideloads receive channel guidance from
  the manual check; they do not use a custom APK installer.

Google Play's own auto-update settings remain under the device owner's control.
The app cannot turn them on. A production draft is not available to ordinary
users: publish an eligible release to the appropriate track before expecting
Play to offer it. Staged rollout eligibility and account ownership also affect
availability.

## Firebase App Distribution testers

The Firebase Fastlane build sets
`MEDIGUIDE_DISTRIBUTION_CHANNEL=firebase`. Only these builds include the full
Firebase App Distribution alert SDK. Play builds explicitly set the channel to
`play`, include only its API library, and reject the full SDK in their runtime
dependency graph.

Before using tester alerts, enable **Firebase App Testers API** in the Google
Cloud project used by the build. A tester taps Profile → Check for Update once
to sign into the prebuilt Firebase tester dialog and enable alerts. Automatic
checks do not open sign-in dialogs for testers who have not opted in. Signed-in
testers receive the SDK's new-build and installation dialogs, with checks
limited to once daily. Manual checks can run sooner. The tester must belong to
the configured distribution group and the new build must be distributed to it.

Build channels are compiled into the APK; do not submit a Firebase tester APK
to Google Play. The existing Play build and draft-upload lanes use the Play
channel. Firebase APK updates require compatible package IDs and signing keys;
switching to Play App Signing may require a separate installation rather than
an in-place replacement.

## Release and verification

1. Build a new mobile version with an increasing Android build number. Existing
   installations acquire this implementation only after installing that build.
2. Upload the Play AAB as usual. Set priorities 4–5 only for deliberately
   approved critical releases; routine releases use priority 0–3.
3. Test using a lower-version Play installation and a higher-version eligible
   internal-track release. Verify availability, decline, download, dismiss,
   background/resume, restart, airplane mode, and relaunch during a download.
4. For Firebase, distribute two increasing tester versions. Verify explicit
   tester sign-in, group eligibility, update alerts, cancellation and offline
   recovery.
5. Verify the Play artifact excludes the full Firebase self-update SDK and the
   tester artifact includes it. Neither flow deletes downloaded clinical data.

`minimum_supported_version` remains reserved and does not block the reader.
Critical update decisions currently use the version actually offered by Play
and its release priority, avoiding a Remote Config minimum that exceeds an
individual user's rollout eligibility.

References: [Play update lifecycle](https://developer.android.com/guide/playcore/in-app-updates/kotlin-java),
[Play update testing](https://developer.android.com/guide/playcore/in-app-updates/test),
[Firebase tester alerts and build isolation](https://firebase.google.com/docs/app-distribution/set-up-alerts?platform=android).

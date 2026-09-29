# MediGuide Google Play release pack

Prepared 28 September 2026 from the inspected repository. Status: NOT READY FOR SUBMISSION.
Package: com.mediguide.ug. Observed mobile version: 2.1.8+60. Flutter: 3.44.8.
Do not assume build 60 is unused; compare against all Play tracks.

## Prepared
- English store listing and release-note drafts in ../fastlane/metadata/android/en-US.
- Production HTTPS enforcement and removal of broad storage/media and battery-exemption permissions.
- Android compile/target SDK floor 36, with no debug signing fallback.
- Fastlane release-signing/HTTPS/flavor checks and a submission preflight.
- Source/metadata checker and export command: python3 tool/check_play_store.py --export.
- Existing production icon reused by the export command; no fabricated screenshots.

## Blocking findings
1. lib/features/profile/presentation/screens/profile_page.dart currently says account deletion is unavailable. Implement an authenticated deletion/request flow with a working backend and identity cleanup, or a functional in-app link to an operational deletion-request web flow. Also publish an accessible external deletion URL. A nonfunctional button or deactivation alone is insufficient.
2. Confirm and publish the privacy policy with the operator's legal name, contact, actual processors, purposes, retention periods, deletion exceptions and AI processing. Existing in-app terms are not proof of a publicly accessible policy.
3. Verify upload-key custody and Play App Signing. No secrets were read, generated, or validated in this review.
4. Produce a signed production AAB and verify the merged manifest, upload certificate, 64-bit libraries, 16 KB native compatibility, and target SDK.
5. Capture actual release-app screenshots and a final 1024x500 feature graphic.
6. Run analysis/tests and device acceptance, including Android 16, offline/PDF access, notifications, AI and account deletion.
7. Finalize Data safety, health-app declaration, content rating, audience, app access and support details.

Remote command execution was not exposed by the repository connector during this review. Flutter/Gradle builds, device tests, signing, console status and live policy URLs have NOT been verified. Source edits are preparation, not a release certification.

## Build locally
From user_app, using the pinned Flutter SDK on PATH and JDK 17:

```sh
flutter pub get
dart run build_runner build --delete-conflicting-outputs
dart format --output=none --set-exit-if-changed lib test
flutter analyze
flutter test
bundle install
python3 tool/check_play_store.py
export MOBILE_FLAVOR=production
export MOBILE_API_BASE_URL=https://mediguide.health.go.ug
# Set MOBILE_BUILD_NUMBER to an unused increasing Play versionCode.
# Set FIREBASE_DART_DEFINES_FILE to your production Firebase defines JSON.
bundle exec fastlane android build_release
python3 tool/check_play_store.py --export
```

Create android/key.properties locally from android/key.properties.example.
Reuse the existing upload key for an existing application. For a genuinely new app,
create an upload key with keytool interactively and back it up securely:
```sh
keytool -genkeypair -v -keystore android/app/upload-keystore.jks -alias upload -keyalg RSA -keysize 2048 -validity 10000
```
Do not commit keys, passwords or service-account files. Confirm the certificate fingerprint
against Play Console; key creation is not required when an upload key already exists.

AAB: build/app/outputs/bundle/productionRelease/app-production-release.aab.
Export directory: build/play-store. This exporter does not build or certify an AAB.

## Play Console steps
1. Use the authorized Ministry/publisher developer account, complete verification and create/select MediGuide.
   Preserve com.mediguide.ug for updates; verify publisher authorization for branding/content.
2. Enable Play App Signing. Record the Play app-signing certificate separately from the upload certificate.
   Register production Play SHA fingerprints with any Firebase/auth integrations that require them.
3. Complete the listing from the drafts. Add verified support contact and published privacy URL;
   upload the existing 512x512 icon, approved 1024x500 feature graphic and real phone screenshots.
4. Complete App content: app access (working reviewer account and instructions), ads, audience,
   content rating, Data safety, health declaration and external account-deletion URL.
5. Build once and manually upload the first signed AAB to internal testing if the app is not yet
   initialized for API uploads. Check Play's bundle diagnostics and pre-launch report.
6. Install from Play internal testing and execute acceptance.csv. Test the upgrade path from any
   existing production version. Resolve crashes, ANRs, broken links and policy warnings.
7. If this is a personal developer account created after 13 November 2023, the production-access
   process requires a closed test with at least 12 opted-in testers for 14 continuous days,
   then an application for production access. Verify the actual account's Console requirements.
8. Merge the verified release changes and use the repository's unified release process:
   make release-prepare RELEASE_TAG=v<next-version>; verify VERSION and every component match.
   Choose an unused mobile build number. Do not move an existing tag.
9. Configure GitHub's production environment variables/secrets below. Run
   "Upload mobile production candidate", choose google-play, select the new stable tag,
   and repeat it in confirmation. The existing workflow uploads a production DRAFT.
   Store listing assets are entered manually; the Fastlane upload deliberately skips metadata.
10. Review the draft in Play Console, resolve all review items, and submit. Use managed publishing
    when coordinating launch. For updates, use a staged rollout; monitor crashes, ANRs and feedback
    before expansion. A first production release may not support staged rollout.

## Existing GitHub configuration
Variables: MOBILE_API_BASE_URL (HTTPS).
Secrets:
- FIREBASE_MOBILE_CONFIG_JSON
- ANDROID_UPLOAD_KEYSTORE_BASE64
- ANDROID_UPLOAD_STORE_PASSWORD
- ANDROID_UPLOAD_KEY_ALIAS
- ANDROID_UPLOAD_KEY_PASSWORD
- GOOGLE_PLAY_SERVICE_ACCOUNT_BASE64

The service account needs access to this app and the release permissions required by the
chosen track; avoid unnecessary account-wide administration. The production workflow builds,
uploads a draft, retains the AAB and removes its temporary credentials.

## Official references (checked 28 September 2026)
- API 36 for new submissions/updates from 31 August 2026:
  https://support.google.com/googleplay/android-developer/answer/11926878
- Native 16 KB compatibility: https://developer.android.com/guide/practices/page-sizes
- Health policy: https://support.google.com/googleplay/android-developer/answer/16679511
- Account deletion: https://support.google.com/googleplay/android-developer/answer/13327111
- Preview assets: https://support.google.com/googleplay/android-developer/answer/9866151
- Personal-account testing: https://support.google.com/googleplay/android-developer/answer/14151465

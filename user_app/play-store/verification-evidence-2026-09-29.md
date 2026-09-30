# Google Play technical verification evidence — 29 September 2026

This record covers technical checks that were actually executed. It does not
replace Play Console review, publisher policy approval or end-to-end acceptance
testing.

## Signed production bundle

- Workflow: `Build Play Store verification bundle`
- GitHub Actions run: <https://github.com/mohuganda/mediguide-pos/actions/runs/36545886164>
- Mobile build number: `61`
- Artifact: `mediguide-play-verification-61`
- Bundle: `app-production-release.aab`
- SHA-256: `cfaa0560534594936eaf78f1542a24cc850689dd99a546d45f3fbcb4fd82e5e2`
- `jarsigner -verify` result: verified
- Upload certificate subject: `CN=MediGuide Android Upload, O=Ministry of Health Uganda, C=UG`
- Upload certificate SHA-1: `64:9A:6F:4C:05:F7:C7:8C:A1:D6:6F:AD:C5:70:DE:6A:AE:26:E8:BD`
- Upload certificate SHA-256: `9D:36:7C:C3:2A:E9:0D:18:7A:73:F3:78:26:1C:24:29:62:C4:4F:95:EF:03:48:96:B8:5D:B1:27:B6:0E:75:FC`
- Certificate expiry: 29 December 2053

On 30 September 2026 the configured Google Play service account successfully
authenticated, accessed `com.mediguide.ug`, read its four visible tracks and
uploaded this AAB to a temporary uncommitted Play edit. Play accepted bundle
version code `61`, proving that the upload certificate matches the application.
The temporary edit was deleted and no release or track change was committed.

The Play app-signing certificate is a different certificate and must still be
recorded separately. The matching `9D:36:...:75:FC` private upload key currently
used by GitHub Actions also needs an approved recoverable backup; the separately
created local keystore has a different certificate and must not replace the
working GitHub secret.

## Android runtime and packaging

- Production profile package `com.mediguide.ug`, version `2.1.8+60`, installed
  and launched on an Android 16 / API 36 arm64 emulator.
- The production profile APK passed `zipalign -c -P 16 -v 4`.
- The app reached the guest home screen and accessed public/cached content.
- This was a launch smoke test. Reader navigation, keyboard/back behavior,
  offline/PDF, notifications, AI, accessibility, upgrade and deletion scenarios
  remain listed in `acceptance.csv`.

## Merged production manifest

The production profile merged manifest was regenerated with:

```sh
cd user_app/android
./gradlew :app:processProductionProfileMainManifest --rerun-tasks
```

Inspection confirmed:

- `targetSdkVersion="36"` and `minSdkVersion="24"`.
- `android:usesCleartextTraffic="false"`.
- no `com.google.android.gms.permission.AD_ID` permission.
- no `android.permission.ACCESS_ADSERVICES_AD_ID` permission.
- no `android.permission.ACCESS_ADSERVICES_ATTRIBUTION` permission.
- `google_analytics_adid_collection_enabled=false`.
- `google_analytics_default_allow_ad_personalization_signals=false`.

## Automated verification

The following passed on 29 September 2026:

- Flutter formatting check for `lib` and `test`.
- `flutter analyze` with no issues.
- Firebase foreground-notification, notification-permission and push-registration
  focused tests.
- Android release-size/privacy regression tests.
- Play Store metadata/source checker.
- Guidelines portal lint, all 71 tests and production build using Node 20.20.1.

## Remaining external evidence

- Play bundle diagnostics and pre-launch report.
- Internal-track installation of the exact signed candidate.
- Completion of all scenarios in `acceptance.csv`.
- Publisher approval of store assets, privacy notice, Data safety and health
  declarations.
- Operational account-deletion fulfilment and retention evidence.

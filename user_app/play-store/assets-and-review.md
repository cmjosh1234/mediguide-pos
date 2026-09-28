# Graphics and reviewer handoff
The exporter reuses android/app/src/production/ic_launcher-playstore.png after verifying
its PNG dimensions are 512x512. A prepared 1024x500 feature graphic is stored at
fastlane/metadata/android/en-US/images/featureGraphic.png. Confirm the final publisher
owns and approves both assets before upload.

Prepared screenshots:
- Six genuine 1080x1920 captures from the production flavor running in profile mode on
  an Android 16 emulator: guest home, empty search, populated search results, guideline
  overview, chapter browser and clinical tools.
- The captures contain no patient-identifiable information and are stored in
  `fastlane/metadata/android/en-US/images/phoneScreenshots`.

Still required:
- Publisher visual approval. Re-capture from the signed production candidate if that build
  renders differently, and add offline-library or AI-citation screens only after those flows
  pass release acceptance with suitable non-sensitive demonstration content.
- Check current Play asset specifications for file size, aspect ratio and transparency.
- Review listing claims against the accepted production build. Text files are drafts.

Reviewer instructions to enter in Play Console:
1. Describe guest access and how to open/search a public guideline.
2. Supply a dedicated working reviewer account for login-only features using the secure Console
   app-access fields (never commit credentials). Explain OTP/SSO access requirements.
3. Give exact paths to offline downloads, AI assistant, privacy policy and account deletion.
4. Describe any geographic, network or role restrictions and how review access is provided.
5. Confirm backend/content availability throughout review.

Do not publish placeholder support emails, URLs or manufactured UI screenshots.

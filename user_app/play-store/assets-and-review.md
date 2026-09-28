# Graphics and reviewer handoff
The exporter reuses android/app/src/production/ic_launcher-playstore.png after verifying
its PNG dimensions are 512x512. Confirm the final publisher owns the artwork.

Still required:
- featureGraphic.png: approved 1024x500 PNG/JPEG, readable at small size.
- At least two actual phone screenshots; prepare six 1080x1920 images covering:
  guest home, guideline search, guideline reader, offline library, clinical tools and AI citations.
  Capture only features present in the release; use demo accounts and no patient identifiers.
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

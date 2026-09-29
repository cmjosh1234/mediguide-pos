import { Link } from "react-router-dom";
import "./legal.css";

export function PrivacyPage() {
  return (
    <article className="page-shell legal-page">
      <p>MediGuide · Privacy</p>
      <h1>Privacy information</h1>
      <p role="status"><strong>Draft for publisher review.</strong> This notice
        requires confirmation of retention periods, service providers and the
        privacy contact before it can be used for the Play Store release.</p>
      <h2>About MediGuide</h2>
      <p>MediGuide provides clinical guidelines and reference tools for health
        workers. The application identifies Ministry of Health Uganda as its
        publisher. Guest access is available for public content; creating an
        account enables personalized features.</p>
      <h2>Information processed</h2>
      <ul>
        <li>Account and professional profile information you provide, such as
          name, email, phone number, facility, organization and professional details.</li>
        <li>Saved notes, bookmarks, reading progress, preferences and support requests.</li>
        <li>Questions, conversation content and retrieved sources used by AI-assisted features.</li>
        <li>Device registration and notification tokens used to deliver app updates.</li>
        <li>Usage events, crash reports and technical diagnostics, including
          identifiers used by Firebase Analytics and Crashlytics.</li>
      </ul>
      <h2>How information is used</h2>
      <p>Information supports account access, saved content, guideline navigation,
        notifications, user support and application reliability. Account-linked
        features send relevant information to MediGuide services. Downloaded
        content and some preferences are stored on your device.</p>
      <h2>Service providers and AI</h2>
      <p>MediGuide integrates Google Firebase services for notifications,
        analytics and crash reporting. AI questions are processed by the
        configured MediGuide AI services. Do not enter patient-identifiable
        or confidential information into questions or notes. AI answers may
        be incorrect; verify cited clinical guidance.</p>
      <h2>Account and data deletion</h2>
      <p>Use <Link to="/delete-account">Request account deletion</Link> or the
        account-deletion option in your mobile profile. Your current password
        verifies ownership. A reference number confirms that the request was
        recorded; it does not mean erasure is complete. The support team must
        process account-linked data and communicate the outcome.</p>
      <h2>Retention and your choices</h2>
      <p>The publisher must finalize and disclose the deletion processing period,
        backup expiry schedule and any specific security or regulatory retention
        exceptions before release. Shared clinical publication records require
        separate assessment from personal profile and activity data.</p>
      <p>You can use supported public content without an account and control
        notification permission in your device settings. Downloaded material can
        become outdated; reconnect to check for updates.</p>
      <h2>Contact</h2>
      <p>Use MediGuide Help &amp; Support for account and privacy questions.
        A verified public privacy contact and effective date must be added to
        the final approved notice.</p>
    </article>
  );
}

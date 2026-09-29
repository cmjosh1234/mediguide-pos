import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { publicApiBaseUrl, dashboardBaseUrl } from "../../config";
import "./legal.css";

export function AccountDeletionPage() {
  const [busy, setBusy] = useState(false);
  const [requestId, setRequestId] = useState("");
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const form = event.currentTarget;
    const fields = new FormData(form);
    setBusy(true);
    setError("");
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), 20000);
    try {
      const base = new URL(publicApiBaseUrl, window.location.origin);
      if (window.location.protocol === "https:" && base.protocol !== "https:") {
        throw new Error("Secure service configuration required.");
      }
      const response = await fetch(
        `${publicApiBaseUrl}/api/v2/auth/deletion-request`,
        {
          method: "POST",
          credentials: "omit",
          cache: "no-store",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            email: fields.get("email"),
            current_password: fields.get("password"),
            confirm: fields.get("confirm") === "on",
          }),
          signal: controller.signal,
        },
      );
      const payload = await response.json();
      if (!response.ok || !payload.success || typeof payload.data?.request_id !== "string") {
        throw new Error(response.status === 429
          ? "Too many attempts. Please wait before trying again."
          : "Unable to submit. Check your account email and current password.");
      }
      form.reset();
      setRequestId(payload.data.request_id);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to submit. Please try again.");
    } finally {
      window.clearTimeout(timeout);
      setBusy(false);
    }
  }

  return (
    <section className="page-shell legal-page">
      <p>MediGuide · Account and data</p>
      <h1>Request account deletion</h1>
      <p>You can request deletion here without installing the mobile app.
        Enter your MediGuide credentials to verify that you own the account.</p>
      <p>The request covers your account and associated personal data, including
        saved notes, reading history and account-linked activity. Support will
        review any shared clinical or security records that require retention
        and confirm the outcome. Sending a request does not immediately erase
        your account or sign you out.</p>
      {requestId ? (
        <div role="status">
          <h2>Request recorded</h2>
          <p>Keep this reference: <strong>{requestId}</strong></p>
          <p>Your account has not yet been deleted. You can follow up through
            MediGuide Help &amp; Support.</p>
        </div>
      ) : (
        <form onSubmit={submit} className="legal-form">
          <label htmlFor="deletion-email">Account email</label>
          <input id="deletion-email" name="email" type="email"
            autoComplete="username" required maxLength={254} disabled={busy} />
          <label htmlFor="deletion-password">Current password</label>
          <input id="deletion-password" name="password" type="password"
            autoComplete="current-password" required maxLength={1024} disabled={busy} />
          <label className="legal-confirm">
            <input name="confirm" type="checkbox" required disabled={busy} />
            I want to request deletion of my account and associated personal data.
          </label>
          {error && <p role="alert">{error}</p>}
          <button className="button" type="submit" disabled={busy}>
            {busy ? "Submitting…" : "Submit deletion request"}
          </button>
          <a href={`${dashboardBaseUrl}/forgot-password`}>Forgot your password?</a>
        </form>
      )}
      <p><Link to="/privacy">Privacy information</Link></p>
    </section>
  );
}

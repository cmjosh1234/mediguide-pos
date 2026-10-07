import { ArrowLeft, ExternalLink, Globe } from "lucide-react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import type { PublicGuideline } from "../../../api/public-guidelines";
import { Brand } from "../../../components/common/Brand";
import { dashboardLoginUrl } from "../../../config";
import { confirmExternalResource } from "../../discovery/resource-navigation";

/** The host a link points to, or "" when it isn't a usable https link. */
function linkHost(url?: string) {
  try {
    const parsed = new URL(url ?? "");
    return parsed.protocol === "https:" ? parsed.hostname : "";
  } catch {
    return "";
  }
}

/**
 * Shows a library document published as a link to an external website. There
 * is nothing to read here, so the page says where the link goes and opens it
 * after the usual external-site confirmation.
 */
export function LinkedDocumentReader({ guideline }: { guideline: PublicGuideline }) {
  const navigate = useNavigate();
  const location = useLocation();
  // A direct visit has no in-app history to return to, so fall back to the library.
  const goBack = () => (location.key === "default" ? navigate("/") : navigate(-1));
  const host = linkHost(guideline.external_url);
  const meta = [guideline.source_org, guideline.version && `Version ${guideline.version}`, guideline.publication_date]
    .filter(Boolean)
    .join(" · ");

  return (
    <div className="backend-reader uploaded-document-reader">
      <header className="backend-reader-header">
        <Link to="/" aria-label="MediGuide guideline library">
          <Brand />
        </Link>
        <nav aria-label="Reader actions">
          <Link to="/">All guidelines</Link>
          <a href={dashboardLoginUrl}>Login</a>
        </nav>
      </header>
      <main id="guideline-content" className="uploaded-document">
        <button type="button" className="uploaded-document-back" onClick={goBack}>
          <ArrowLeft size={18} aria-hidden="true" />
          Back
        </button>
        <div className="uploaded-document-heading">
          <div>
            {guideline.document_kind?.name && (
              <p className="uploaded-document-kind">{guideline.document_kind.name}</p>
            )}
            <h1>{guideline.title}</h1>
            {meta && <p className="uploaded-document-meta">{meta}</p>}
            {guideline.description && <p>{guideline.description}</p>}
          </div>
        </div>
        {host ? (
          <section className="linked-document" aria-label="External website">
            <span className="linked-document-icon" aria-hidden="true">
              <Globe size={22} />
            </span>
            <div>
              <h2>External website</h2>
              <p className="linked-document-host">{host}</p>
              <p>
                This resource is published as a link. It opens in a new tab, and its content is maintained by the
                website that publishes it.
              </p>
              <a
                className="button button-primary"
                href={guideline.external_url}
                target="_blank"
                rel="noreferrer"
                onClick={(event) => {
                  if (!confirmExternalResource()) event.preventDefault();
                }}
              >
                Open website <ExternalLink size={14} aria-hidden="true" />
              </a>
            </div>
          </section>
        ) : (
          <p className="uploaded-document-status" role="alert">
            This link is unavailable.
          </p>
        )}
      </main>
    </div>
  );
}

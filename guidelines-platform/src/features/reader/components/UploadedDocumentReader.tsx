import { ArrowLeft } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import {
  getPublicGuidelineOriginal,
  getPublicGuidelineOriginalFile,
  type PublicGuideline,
  type PublicGuidelineAssetLink,
} from "../../../api/public-guidelines";
import { Brand } from "../../../components/common/Brand";
import { dashboardLoginUrl } from "../../../config";

export type UploadedFileState =
  | { status: "loading" }
  | { status: "ready"; url: string; asset: PublicGuidelineAssetLink }
  | { status: "error" };

/**
 * Shows a guideline published as its uploaded file (for example a form),
 * exactly as it was uploaded: PDFs inline, Word files as a download.
 * Render it with `key={guideline.id}` so another guideline starts from loading.
 */
export function UploadedDocumentReader({
  guideline,
  loadOriginal = getPublicGuidelineOriginal,
  loadFile = getPublicGuidelineOriginalFile,
}: {
  guideline: PublicGuideline;
  loadOriginal?: typeof getPublicGuidelineOriginal;
  loadFile?: typeof getPublicGuidelineOriginalFile;
}) {
  const [file, setFile] = useState<UploadedFileState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    let objectUrl: string | undefined;
    loadOriginal(guideline.id, controller.signal)
      .then(async (asset) => {
        const blob = await loadFile(asset, controller.signal);
        objectUrl = URL.createObjectURL(blob);
        setFile({ status: "ready", url: objectUrl, asset });
      })
      .catch(() => {
        if (!controller.signal.aborted) setFile({ status: "error" });
      });
    return () => {
      controller.abort();
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [guideline.id, loadOriginal, loadFile]);

  return <UploadedDocumentView guideline={guideline} file={file} />;
}

/** The presentation of an uploaded document for a given file state. */
export function UploadedDocumentView({
  guideline,
  file,
}: {
  guideline: PublicGuideline;
  file: UploadedFileState;
}) {
  const navigate = useNavigate();
  const location = useLocation();
  // A direct visit has no in-app history to return to, so fall back to the library.
  const goBack = () => (location.key === "default" ? navigate("/") : navigate(-1));
  const isPdf = file.status === "ready" && file.asset.mime_type === "application/pdf";
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
          {file.status === "ready" && (
            <a
              className="button button-primary"
              href={file.url}
              download={file.asset.original_filename || guideline.slug}
            >
              Download {isPdf ? "PDF" : "Word document"}
            </a>
          )}
        </div>
        {file.status === "loading" && (
          <p className="uploaded-document-status">Loading the document…</p>
        )}
        {file.status === "error" && (
          <p className="uploaded-document-status" role="alert">
            The document could not be loaded. Check your connection and try again.
          </p>
        )}
        {file.status === "ready" &&
          (isPdf ? (
            <iframe
              className="uploaded-document-frame"
              title={`${guideline.title} (PDF)`}
              src={file.url}
            />
          ) : (
            <p className="uploaded-document-status">
              This document is a Word file. Download it to view and print it exactly as published.
            </p>
          ))}
      </main>
    </div>
  );
}

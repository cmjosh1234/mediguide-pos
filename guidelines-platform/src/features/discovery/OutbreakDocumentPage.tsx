import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import {
  getPublicOutbreakDocument,
  getPublicOutbreakDocumentContent,
  PublicApiError,
  resolvePublicAssetUrl,
  type PublicOutbreakDocument,
  type PublicOutbreakDocumentContent,
} from "../../api/public-guidelines";
import { SecureMarkdown } from "../reader/components/SecureMarkdown";
import { LoadingCards, OfflineNotice, StateMessage } from "./DiscoveryComponents";
import { dateLabel, decodeSanitizedMarkdown } from "./discovery-utils";

type LoadState =
  | { status: "loading" }
  | { status: "not-found" }
  | { status: "error" }
  | {
      status: "ready";
      document: PublicOutbreakDocument;
      content?: PublicOutbreakDocumentContent;
      offline?: boolean;
    };

function formatBytes(bytes?: number) {
  if (!bytes) return undefined;
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function OutbreakDocumentPage() {
  const { outbreakId = "", documentId = "" } = useParams();
  const navigate = useNavigate();
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<LoadState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    getPublicOutbreakDocument(outbreakId, documentId, controller.signal)
      .then(async (document) => {
        const content = document.supports_inline
          ? await getPublicOutbreakDocumentContent(
              documentId,
              controller.signal,
            ).catch(() => undefined)
          : undefined;
        if (controller.signal.aborted) return;
        setState({
          status: "ready",
          document,
          content,
          offline: document.offline || content?.offline,
        });
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setState({
          status:
            error instanceof PublicApiError && error.kind === "not-found"
              ? "not-found"
              : "error",
        });
      });
    return () => controller.abort();
  }, [outbreakId, documentId, attempt]);

  const back = (
    <button
      type="button"
      className="outbreak-doc-back"
      onClick={() =>
        window.history.length > 1 ? navigate(-1) : navigate("/diseases")
      }
    >
      ← Back
    </button>
  );

  if (state.status === "loading")
    return (
      <section className="page-shell discovery-page">
        <LoadingCards />
      </section>
    );
  if (state.status !== "ready")
    return (
      <section className="page-shell discovery-page">
        {back}
        <StateMessage
          title={
            state.status === "not-found"
              ? "This outbreak document is not published or has been withdrawn."
              : "This outbreak document could not be loaded."
          }
          retry={
            state.status === "error"
              ? () => {
                  setState({ status: "loading" });
                  setAttempt((value) => value + 1);
                }
              : undefined
          }
        />
      </section>
    );

  const { document, content } = state;
  const facts: Array<[string, string | undefined]> = [
    ["Issued by", document.issuing_authority],
    ["Document no.", document.document_number],
    ["Version", document.version],
    ["Audience", document.audience],
    ["Effective", document.effective_date && dateLabel(document.effective_date)],
    ["Review by", document.review_date && dateLabel(document.review_date)],
  ];
  const size = formatBytes(document.file_size);

  return (
    <section className="page-shell discovery-page outbreak-doc">
      {state.offline && <OfflineNotice />}
      {back}
      <span className="eyebrow">
        {document.document_kind?.replace(/_/g, " ") || "Outbreak document"}
      </span>
      <h1>{document.title}</h1>
      {document.description && <p>{document.description}</p>}
      {document.outbreak_title && (
        <p className="outbreak-doc-context">
          Part of the <strong>{document.outbreak_title}</strong>
          {document.outbreak_area && <> · {document.outbreak_area}</>}
        </p>
      )}
      <dl className="outbreak-doc-facts">
        {facts
          .filter(([, value]) => value)
          .map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
      </dl>
      {document.download_url && (
        <p>
          <a
            className="button button-primary"
            href={resolvePublicAssetUrl(document.download_url)}
            download={document.original_filename}
          >
            Download original{size ? ` (${size})` : ""}
          </a>
        </p>
      )}
      {content?.content ? (
        <article className="markdown-content outbreak-doc-body">
          <SecureMarkdown content={decodeSanitizedMarkdown(content.content)} />
        </article>
      ) : (
        <StateMessage title="This document has no inline preview. Download the original to read it." />
      )}
      <p>
        <Link to="/diseases">Browse all diseases and conditions</Link>
      </p>
    </section>
  );
}

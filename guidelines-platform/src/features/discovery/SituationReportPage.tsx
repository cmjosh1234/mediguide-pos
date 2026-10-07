import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { CheckCircle2, Download, ExternalLink } from "lucide-react";

import {
  getPublicSituationReport,
  PublicApiError,
  resolvePublicAssetUrl,
  type PublicResource,
  type PublicSituationReport,
} from "../../api/public-guidelines";
import "../../styles/situation-report.css";
import { LoadingCards, OfflineNotice, ResourceGrid, StateMessage } from "./DiscoveryComponents";
import { dateLabel, metricValueLabel } from "./discovery-utils";
import { confirmExternalResource, isExternalResourceRoute } from "./resource-navigation";

type LoadState =
  | { status: "loading" }
  | { status: "not-found" }
  | { status: "error" }
  | { status: "ready"; report: PublicSituationReport; offline?: boolean };

// Attachments link /public/guidelines/{id}; the portal reads them at /guidelines/{id}.
function attachmentResources(report: PublicSituationReport): PublicResource[] {
  return [...(report.attachments ?? [])]
    .sort((a, b) => a.sort_order - b.sort_order)
    .map((attachment) => {
      const documentId = attachment.url.match(/^\/public\/guidelines\/([0-9a-fA-F-]{36})$/)?.[1];
      return {
        id: attachment.id,
        content_type: attachment.document_kind || "guideline",
        title: attachment.title,
        description: attachment.description,
        issuing_authority: attachment.issuing_organization,
        route: documentId ? `/guidelines/${documentId}` : undefined,
      };
    });
}

function BackButton() {
  const navigate = useNavigate();
  return (
    <button
      type="button"
      className="outbreak-doc-back"
      onClick={() => (window.history.length > 1 ? navigate(-1) : navigate("/diseases"))}
    >
      ← Back
    </button>
  );
}

/** A published situation report: its key figures, highlights and documents. */
export function SituationReportPage() {
  const { reportId = "" } = useParams();
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<LoadState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    getPublicSituationReport(reportId, controller.signal)
      .then((report) => setState({ status: "ready", report, offline: report.offline }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setState({ status: error instanceof PublicApiError && error.kind === "not-found" ? "not-found" : "error" });
      });
    return () => controller.abort();
  }, [reportId, attempt]);

  if (state.status === "loading")
    return (
      <section className="page-shell discovery-page">
        <LoadingCards />
      </section>
    );
  if (state.status !== "ready")
    return (
      <section className="page-shell discovery-page">
        <BackButton />
        <StateMessage
          title={
            state.status === "not-found"
              ? "This situation report is not published or has been withdrawn."
              : "This situation report could not be loaded."
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

  return <SituationReportView report={state.report} offline={state.offline} />;
}

export function SituationReportView({ report, offline }: { report: PublicSituationReport; offline?: boolean }) {
  const facts: Array<[string, string | undefined]> = [
    ["Issued by", report.source_organization],
    ["Area", report.geographic_area],
    ["Published", report.publication_date && dateLabel(report.publication_date)],
    ["Effective", report.effective_at && dateLabel(report.effective_at)],
    ["Data as of", report.data_as_of && dateLabel(report.data_as_of)],
    ["Last verified", report.last_verified_at && dateLabel(report.last_verified_at)],
    ["Source reference", report.source_reference],
  ];
  const metrics = [...(report.metrics ?? [])].sort((a, b) => a.sort_order - b.sort_order);
  const highlights = report.key_highlights ?? [];
  const documents = attachmentResources(report);
  // With attachments, report_asset_url only repeats the first one for older apps.
  const pdf = documents.length === 0 && report.report_asset_url ? report.report_asset_url : undefined;
  const pdfIsExternal = isExternalResourceRoute(pdf);

  return (
    <section className="page-shell discovery-page situation-report">
      {offline && <OfflineNotice />}
      <BackButton />
      <span className="eyebrow">Situation report</span>
      <h1>{report.title}</h1>
      {report.summary && <p>{report.summary}</p>}
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
      {(pdf || report.source_url) && (
        <div className="situation-report-actions">
          {pdf && (
            <a
              className="button button-primary"
              href={pdfIsExternal ? pdf : resolvePublicAssetUrl(pdf)}
              target="_blank"
              rel="noreferrer"
              onClick={(event) => {
                if (pdfIsExternal && !confirmExternalResource()) event.preventDefault();
              }}
            >
              <Download aria-hidden="true" size={16} /> Download full report
            </a>
          )}
          {report.source_url && (
            <a
              className="button"
              href={report.source_url}
              target="_blank"
              rel="noreferrer"
              onClick={(event) => {
                if (!confirmExternalResource()) event.preventDefault();
              }}
            >
              View source <ExternalLink aria-hidden="true" size={14} />
            </a>
          )}
        </div>
      )}
      {metrics.length > 0 && (
        <>
          <h2>Key figures</h2>
          <div className="metrics-grid situation-report-metrics">
            {metrics.map((metric) => (
              <div className="metric-tile" key={metric.key}>
                <span className="metric-label">{metric.label}</span>
                <strong className="metric-value">
                  {metricValueLabel(metric)}
                  {metric.unit && <small> {metric.unit}</small>}
                </strong>
                <span className="metric-source">
                  {[metric.source_reference, metric.as_of && `as of ${dateLabel(metric.as_of)}`].filter(Boolean).join(" · ")}
                </span>
              </div>
            ))}
          </div>
        </>
      )}
      {highlights.length > 0 && (
        <>
          <h2>Key highlights</h2>
          <ul className="situation-report-highlights">
            {highlights.map((highlight) => (
              <li key={highlight}>
                <CheckCircle2 aria-hidden="true" size={18} />
                <span>{highlight}</span>
              </li>
            ))}
          </ul>
        </>
      )}
      {documents.length > 0 && (
        <>
          <h2>Report documents</h2>
          <ResourceGrid resources={documents} />
        </>
      )}
      <p>
        <Link to="/diseases">Browse all diseases and conditions</Link>
      </p>
    </section>
  );
}

"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { showToast } from "@/lib/toast";
import { outbreaksService, type OutbreakAuditRecord, type PagedResult } from "@/services/outbreaks.service";
import { situationReportsService } from "@/services/situation-reports.service";

import { DEFAULT_PAGE_SIZE, PaginationControls } from "./list-pagination";

const SERVICES = { outbreak: outbreaksService, situation_report: situationReportsService };

const ACTIONS: Record<string, string> = {
  created: "Created",
  updated: "Edited",
  deleted: "Deleted",
  submit: "Submitted for review",
  approve: "Approved",
  publish: "Published",
  withdraw: "Withdrawn",
  update_status: "Status changed",
  correction_created: "Correction started",
  correction_applied: "Correction applied",
  metrics_updated: "Metrics updated",
  review_comment: "Review comment",
  attachment_added: "Attachment added",
  attachment_removed: "Attachment removed",
  asset_replaced: "Report file replaced",
};

/**
 * The review comment box and the paged audit trail of an outbreak or a
 * situation report: what was done, by whom, when and why. Reloads from the
 * first page whenever refreshKey changes.
 */
export function AuditHistory({ entity, id, refreshKey }: { entity: keyof typeof SERVICES; id: string; refreshKey?: unknown }) {
  const service = SERVICES[entity];
  const [page, setPage] = React.useState(1);
  const [reload, setReload] = React.useState(0);
  const [history, setHistory] = React.useState<PagedResult<OutbreakAuditRecord> | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState("");
  const [comment, setComment] = React.useState("");
  const [posting, setPosting] = React.useState(false);

  React.useEffect(() => setPage(1), [refreshKey]);
  React.useEffect(() => {
    let current = true;
    setLoading(true);
    setError("");
    service
      .audit(id, page, DEFAULT_PAGE_SIZE)
      .then((result) => { if (current) setHistory(result); })
      .catch((value) => { if (current) setError(value instanceof Error ? value.message : "Unable to load the audit history"); })
      .finally(() => { if (current) setLoading(false); });
    return () => { current = false; };
  }, [service, id, page, refreshKey, reload]);

  async function addComment() {
    setPosting(true);
    try {
      await service.addReviewComment(id, comment.trim());
      setComment("");
      setPage(1);
      setReload((value) => value + 1);
      showToast.success("Review comment added", "The comment is recorded in audit history.");
    } catch (value) {
      showToast.error("Comment not added", value instanceof Error ? value.message : undefined);
    } finally {
      setPosting(false);
    }
  }

  const items = history?.items || [];
  const total = history?.total_items || 0;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Review and audit history</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-2">
          <Textarea maxLength={4000} value={comment} onChange={(event) => setComment(event.target.value)} placeholder="Add an auditable reviewer comment" disabled={posting} />
          <Button variant="outline" disabled={posting || !comment.trim()} onClick={() => void addComment()}>
            Add comment
          </Button>
        </div>
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        {loading && !history ? (
          <p className="py-6 text-center text-sm text-muted-foreground"><Loader2 className="mr-2 inline h-4 w-4 animate-spin" />Loading audit history…</p>
        ) : null}
        {history && !items.length ? <p className="text-sm text-muted-foreground">No audit events recorded.</p> : null}
        {items.length ? (
          <ol className={`space-y-2 ${loading ? "opacity-60" : ""}`} aria-label="Audit history" aria-busy={loading}>
            {items.map((entry) => <AuditEntry key={entry.id} entry={entry} />)}
          </ol>
        ) : null}
        <PaginationControls
          label="Audit history"
          pagination={{
            visible: items,
            offset: (page - 1) * DEFAULT_PAGE_SIZE,
            page,
            pageCount: Math.max(1, history?.total_pages || 1),
            pageSize: DEFAULT_PAGE_SIZE,
            total,
            setPage,
            showIndex: (index) => setPage(Math.floor(index / DEFAULT_PAGE_SIZE) + 1),
          }}
        />
      </CardContent>
    </Card>
  );
}

function AuditEntry({ entry }: { entry: OutbreakAuditRecord }) {
  const details = auditDetails(entry);
  return (
    <li className="rounded-md border p-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="font-medium">{actionLabel(entry)}</p>
        <time dateTime={entry.created_at} className="text-xs text-muted-foreground">{new Date(entry.created_at).toLocaleString()}</time>
      </div>
      <p className="text-sm text-muted-foreground" title={entry.actor_id ? `User ID ${entry.actor_id}` : undefined}>
        by <span className="font-medium text-foreground">{actorName(entry)}</span>
        {entry.actor_email ? ` · ${entry.actor_email}` : null}
      </p>
      {details.length ? (
        <dl className="mt-2 grid gap-1 text-sm sm:grid-cols-[max-content_1fr] sm:gap-x-3">
          {details.map(([term, value]) => (
            <React.Fragment key={term}>
              <dt className="text-muted-foreground">{term}</dt>
              <dd className="break-words">{value}</dd>
            </React.Fragment>
          ))}
        </dl>
      ) : null}
    </li>
  );
}

export function actionLabel(entry: OutbreakAuditRecord) {
  const verb = entry.action.slice(entry.action.indexOf(".") + 1);
  if (verb === "approve" && entry.metadata?.applied_to) return "Approved and applied";
  return ACTIONS[verb] || sentence(verb);
}

function actorName(entry: OutbreakAuditRecord) {
  if (entry.actor_name) return entry.actor_name;
  if (!entry.actor_id || /^0{8}-/.test(entry.actor_id)) return "System";
  return `Unknown user (${entry.actor_id.slice(0, 8)})`;
}

/** The reason, comment and changes recorded with an entry, as label and value pairs. */
export function auditDetails(entry: OutbreakAuditRecord): Array<[string, string]> {
  const meta = entry.metadata || {};
  const name = (value: unknown) => (typeof value === "string" && entry.labels?.[value]) || display(value);
  const details: Array<[string, string]> = [];
  if (typeof meta.from_status === "string" && typeof meta.to_status === "string") {
    details.push(["Status", `${sentence(meta.from_status)} → ${sentence(meta.to_status)}`]);
  }
  if (typeof meta.reason === "string" && meta.reason.trim()) details.push(["Reason", meta.reason.trim()]);
  if (typeof meta.comment === "string" && meta.comment.trim()) details.push(["Comment", meta.comment.trim()]);
  if (typeof meta.corrected_by === "string") details.push(["Correction by", name(meta.corrected_by)]);
  if (meta.changes && typeof meta.changes === "object") {
    for (const [field, change] of Object.entries(meta.changes as Record<string, { from?: unknown; to?: unknown }>)) {
      details.push([sentence(field.replace(/_id$/, "")), `${name(change?.from)} → ${name(change?.to)}`]);
    }
  }
  if (entry.action.endsWith("attachment_added") && typeof meta.title === "string") details.push(["Attachment", meta.title]);
  return details;
}

function display(value: unknown) {
  if (value === null || value === undefined || value === "") return "(empty)";
  const text = typeof value === "string" ? value : JSON.stringify(value);
  return text.length > 160 ? `${text.slice(0, 157)}…` : text;
}

function sentence(value: string) {
  const text = value.replace(/_/g, " ");
  return text.charAt(0).toUpperCase() + text.slice(1);
}
